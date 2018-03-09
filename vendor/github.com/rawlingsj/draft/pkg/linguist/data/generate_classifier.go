// +build ignore

/*
   This program trains a naive bayesian classifier
   provided by https://github.com/jbrukh/bayesian
   on a set of source code files
   provided by https://github.com/github/linguist

   This file is meant by run by go generate,
   refer to generate.go for its intended invokation
*/
package main

import (
	"container/heap"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/Azure/draft/pkg/linguist/tokenizer"
	"github.com/jbrukh/bayesian"
)

type sampleFile struct {
	lang, fp string
	tokens   []string
}

func main() {
	const (
		sourcePath = "./linguist/samples"
		outfile    = "./classifier"
		quiet      = false
	)

	log.SetFlags(0)
	if quiet {
		log.SetOutput(ioutil.Discard)
	}

	// first we only read all the paths of the sample files
	// and their corresponding and language names into:
	sampleFiles := []*sampleFile{}
	// and store all the language names into:
	languages := []string{}

	/*
			   github/linguist has directory structure:

			   ...
			   ├── samples
			   │   ├── (name of programming language)
			   │   │   ├── (sample file in language)
			   │   │   ├── (sample file in language)
			   │   │   └── (sample file in language)
			   │   ├── (name of another programming language)
			   │   │   └── (sample file)
			   ...

		       the following hard-coded logic expects this layout
	*/

	log.Println("Scanning", sourcePath, "...")
	srcDir, err := os.Open(sourcePath)
	checkErr(err)

	subDirs, err := srcDir.Readdir(-1)
	checkErr(err)

	for _, langDir := range subDirs {
		lang := langDir.Name()
		if !langDir.IsDir() {
			log.Println("unexpected file:", lang)
			continue
		}

		languages = append(languages, lang)

		samplePath := sourcePath + "/" + lang
		sampleDir, err := os.Open(samplePath)
		checkErr(err)
		files, err := sampleDir.Readdir(-1)
		checkErr(err)
		for _, file := range files {
			fp := samplePath + "/" + file.Name()
			if file.IsDir() {
				// Skip subdirectories
				continue
			}
			sampleFiles = append(sampleFiles, &sampleFile{lang, fp, nil})
		}
		sampleDir.Close()
	}
	log.Println("Found", len(languages), "languages in", len(sampleFiles), "files")

	// simple progress bar
	progress := 0.0
	total := float64(len(sampleFiles)) * 2.0
	progressBar := func() {
		progress++
		fmt.Printf("Processing files ... %.2f%%\r", progress/total*100.0)
	}

	// then we concurrently read and tokenize the samples
	sampleChan := make(chan *sampleFile)
	readyChan := make(chan struct{})
	received := 0
	tokenize := func(s *sampleFile) {
		f, err := os.Open(s.fp)
		checkErr(err)
		contents, err := ioutil.ReadAll(f)
		f.Close()
		checkErr(err)
		s.tokens = tokenizer.Tokenize(contents)
		sampleChan <- s
	}
	dox := map[string][]string{}
	for _, lang := range languages {
		dox[lang] = []string{}
	}
	// this receives the processed files and stores their tokens with their language
	go func() {
		for {
			s := <-sampleChan
			dox[s.lang] = append(dox[s.lang], s.tokens...)
			received++
			progressBar()
			if received == len(sampleFiles) {
				close(readyChan)
				return
			}
		}
	}()

	// this balances the workload (implementation at end of file)
	requests := getRequestsChan(len(sampleFiles))
	for i := range sampleFiles {
		requests <- &request{
			workFn: tokenize,
			arg:    sampleFiles[i],
		}
		progressBar()
	}

	// once that's done
	<-readyChan
	close(requests)
	fmt.Println() // for the progress bar

	// we train the classifier in the arbitrary manner that its API demands
	classes := make([]bayesian.Class, 1)
	documents := make(map[bayesian.Class][]string)
	for _, lang := range languages {
		var class = bayesian.Class(lang)
		classes = append(classes, class)
		documents[class] = dox[lang]
	}
	log.Println("Creating bayesian.Classifier ...")
	clsf := bayesian.NewClassifier(classes...)
	for cls, dox := range documents {
		clsf.Learn(dox, cls)
	}

	// and write the data to disk
	log.Println("Serializing and exporting bayesian.Classifier to", outfile, "...")
	checkErr(clsf.WriteToFile("classifier"))

	log.Println("Done.")
}
func checkErr(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

// simple load balancer from "concurrency is not parallelism" talk
type request struct {
	workFn func(s *sampleFile)
	arg    *sampleFile
}
type worker struct {
	requests       chan *request
	pending, index int
}

func (w *worker) work(done chan *worker) {
	for {
		req := <-w.requests
		req.workFn(req.arg)
		done <- w
	}
}

type pool []*worker

func (p pool) Less(i, j int) bool  { return p[i].pending < p[j].pending }
func (p pool) Len() int            { return len(p) }
func (p pool) Swap(i, j int)       { p[i], p[j] = p[j], p[i] }
func (p *pool) Push(x interface{}) { *p = append(*p, x.(*worker)) }
func (p *pool) Pop() interface{} {
	old := *p
	n := len(old)
	x := old[n-1]
	*p = old[0 : n-1]
	return x
}

type balancer struct {
	workers pool
	done    chan *worker
}

func (b *balancer) balance(work chan *request) {
	for {
		select {
		case req, ok := <-work:
			if ok {
				b.dispatch(req)
			} else {
				return
			}
		case w := <-b.done:
			b.completed(w)
		}
	}
}
func (b *balancer) dispatch(req *request) {
	w := heap.Pop(&b.workers).(*worker)
	w.requests <- req
	w.pending++
	heap.Push(&b.workers, w)
}
func (b *balancer) completed(w *worker) {
	w.pending--
	heap.Remove(&b.workers, w.index)
	heap.Push(&b.workers, w)
}
func getRequestsChan(jobs int) chan *request {
	done := make(chan *worker)
	workers := make(pool, runtime.GOMAXPROCS(0)*4) // I don't know how many workers there should be
	for i := 0; i < len(workers); i++ {
		w := &worker{make(chan *request, jobs), 0, i}
		go w.work(done)
		workers[i] = w
	}
	heap.Init(&workers)
	b := &balancer{workers, done}
	requests := make(chan *request)
	go b.balance(requests)
	return requests
}
