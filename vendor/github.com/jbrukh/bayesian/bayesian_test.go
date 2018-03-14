package bayesian

import "testing"
import "fmt"
import "os"

const (
	Good Class = "good"
	Bad  Class = "bad"
)

func Assert(t *testing.T, condition bool, args ...interface{}) {
	if !condition {
		t.Fatal(args)
	}
}

func TestEmpty(t *testing.T) {
	c := NewClassifier("Good", "Bad", "Neutral")
	priors := c.getPriors()
	for _, item := range priors {
		Assert(t, item == 0)
	}
}

func TestNoClasses(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			// we are good
		}
	}()
	c := NewClassifier()
	Assert(t, false, "should have panicked:", c)
}

func TestNotUnique(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			// we are good
		}
	}()
	c := NewClassifier("Good", "Good", "Bad", "Cow")
	Assert(t, false, "should have panicked:", c)
}

func TestOneClass(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			// we are good
		}
	}()
	c := NewClassifier(Good)
	Assert(t, false, "should have panicked:", c)
}

func TestObserve(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Observe("tall", 2, Good)
	c.Observe("handsome", 1, Good)
	c.Observe("rich", 1, Good)
	c.Observe("bald", 1, Bad)
	c.Observe("poor", 2, Bad)
	c.Observe("ugly", 1, Bad)

	score, likely, strict := c.LogScores([]string{"the", "tall", "man"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] > score[1], "not good, round 1") // this is good
	Assert(t, likely == 0, "not good, round 1")
	Assert(t, strict == true, "not strict, round 1")

	score, likely, strict = c.LogScores([]string{"poor", "ugly", "girl"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] < score[1]) // this is bad
	Assert(t, likely == 1)
	Assert(t, strict == true)

	score, likely, strict = c.LogScores([]string{"the", "bad", "man"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] == score[1], "not the same") // same
	Assert(t, likely == 0, "not good")              // first one is picked
	Assert(t, strict == false, "not strict")
}

func TestLearn(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"bald", "poor", "ugly"}, Bad)

	score, likely, strict := c.LogScores([]string{"the", "tall", "man"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] > score[1], "not good, round 1") // this is good
	Assert(t, likely == 0, "not good, round 1")
	Assert(t, strict == true, "not strict, round 1")

	score, likely, strict = c.LogScores([]string{"poor", "ugly", "girl"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] < score[1]) // this is bad
	Assert(t, likely == 1)
	Assert(t, strict == true)

	score, likely, strict = c.LogScores([]string{"the", "bad", "man"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] == score[1], "not the same") // same
	Assert(t, likely == 0, "not good")              // first one is picked
	Assert(t, strict == false, "not strict")
}

func TestProbScores(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"bald", "poor", "ugly"}, Bad)

	score, likely, strict := c.ProbScores([]string{"the", "tall", "man"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] > score[1], "not good, round 1") // this is good
	Assert(t, likely == 0, "not good, round 1")
	Assert(t, strict == true, "not strict, round 1")

	score, likely, strict = c.ProbScores([]string{"poor", "ugly", "girl"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] < score[1]) // this is bad
	Assert(t, likely == 1)
	Assert(t, strict == true)

	score, likely, strict = c.ProbScores([]string{"the", "bad", "man"})
	fmt.Printf("%v\n", score)
	Assert(t, score[0] == score[1], "not the same") // same
	Assert(t, likely == 0, "not good")              // first one is picked
	Assert(t, strict == false, "not strict")
}

func TestSeenLearned(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"bald", "poor", "ugly"}, Bad)
	doc1 := []string{"hehe"}
	doc2 := []string{}
	doc3 := []string{"ayaya", "ppo", "lim", "inf"}
	var scores []float64
	scores, _, _ = c.LogScores(doc1)
	scores, _, _ = c.LogScores(doc2)
	scores, _, _ = c.LogScores(doc3)
	scores, _, _ = c.ProbScores(doc1)
	scores, _, _ = c.ProbScores(doc2)
	scores, _, _ = c.ProbScores(doc3)
	scores, _, _, _ = c.SafeProbScores(doc1)
	scores, _, _, _ = c.SafeProbScores(doc2)
	scores, _, _, _ = c.SafeProbScores(doc3)
	println(scores)
	Assert(t, c.Learned() == 2, "learned")
	Assert(t, c.Seen() == 9, "seen")
	count := c.WordCount()
	Assert(t, count[0] == 3, "counted-good")
	Assert(t, count[1] == 3, "counted-bad")
	Assert(t, c.Learned() == 2, "learned")

}

func TestInduceUnderflow(t *testing.T) {
	c := NewClassifier(Good, Bad) // knows no words
	const DOC_SIZE = 1000
	document := make([]string, DOC_SIZE)
	for i := 0; i < DOC_SIZE; i++ {
		document[i] = "word"
	}
	// should induce overflow, because each word
	// will have "defaultProb", which is small
	scores, _, _, err := c.SafeProbScores(document)
	Assert(t, err == ErrUnderflow, "Underflow error not detected")
	println(scores)
}

func TestLogScores(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	data := c.datas[Good]
	Assert(t, data.Total == 3)
	Assert(t, data.getWordProb("tall") == float64(1)/float64(3), "tall")
	Assert(t, data.getWordProb("rich") == float64(1)/float64(3), "rich")
	Assert(t, c.WordCount()[0] == 3)
}

func TestGobs(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	err := c.WriteToFile("test.ser")
	Assert(t, err == nil, "could not write:", err)
	d, err := NewClassifierFromFile("test.ser")
	Assert(t, err == nil, "could not read:", err)
	fmt.Printf("%v\n", d)
	scores, _, _ := d.LogScores([]string{"a", "b", "c"})
	println(scores)
	data := d.datas[Good]
	Assert(t, data.Total == 3)
	Assert(t, data.getWordProb("tall") == float64(1)/float64(3), "tall")
	Assert(t, data.getWordProb("rich") == float64(1)/float64(3), "rich")
	Assert(t, d.Learned() == 1)
	count := d.WordCount()
	Assert(t, count[0] == 3)
	Assert(t, count[1] == 0)
	Assert(t, d.Seen() == 1)
	// remove the file
	err = os.Remove("test.ser")
	Assert(t, err == nil, "could not remove test file:", err)
}

func TestClassByFile(t *testing.T) {
	c := NewClassifier(Good, Bad)
	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	err := c.WriteClassesToFile(".")
	Assert(t, err == nil, "could not write class:", err)

	d := NewClassifier(Good, Bad)
	err = d.ReadClassFromFile(Good, ".")
	Assert(t, err == nil, "could not read:", err)
	fmt.Printf("%v\n", d)
	scores, _, _ := d.LogScores([]string{"a", "b", "c"})
	println(scores)
	data := d.datas[Good]
	Assert(t, data.Total == 3)
	Assert(t, data.getWordProb("tall") == float64(1)/float64(3), "tall")
	Assert(t, data.getWordProb("rich") == float64(1)/float64(3), "rich")
	Assert(t, d.Learned() == 1, "learned")
	count := d.WordCount()

	Assert(t, count[0] == 3)
	Assert(t, count[1] == 0)
	Assert(t, d.Seen() == 1)
	// remove the file
	err = os.Remove("good")
	Assert(t, err == nil, "could not remove test file:", err)
	err = os.Remove("bad")
	Assert(t, err == nil, "could not remove test file:", err)
}

func TestFreqMatrixConstruction(t *testing.T) {
	c := NewClassifier(Good, Bad)
	freqs := c.WordFrequencies([]string{"a", "b"})
	Assert(t, len(freqs) == 2, "size")
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			Assert(t, freqs[i][j] == defaultProb, i, j)
		}
	}
}

func TestTfIdClassifier_SanityChecks(t *testing.T) {
	c := NewClassifierTfIdf(Good, Bad)
	Assert(t, c.IsTfIdf() == true)

	c.Learn([]string{"tall", "handsome", "rich"}, Good)

	defer func() {
		if err := recover(); err != nil {
			// we are good
		}
	}()
	c.LogScores([]string{"a", "b", "c"})
	Assert(t, false, "Should have panicked:Need to run ConvertTermsFreqToTfIdf() first..", c)

}

func TestTfIdClassifier_Tf_Checks(t *testing.T) {
	c := NewClassifierTfIdf(Good, Bad)
	Assert(t, c.IsTfIdf() == true)

	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"tall", "blonde"}, Good)
	c.Learn([]string{"tall"}, Good)

	data := c.datas[Good]
	// Total words seen in training.
	Assert(t, data.Total == 6)

	// Plain old counts for words.
	Assert(t, data.Freqs["tall"] == 3)
	Assert(t, data.Freqs["blonde"] == 1)

	// Check for term frequency's per 'document' (tall)
	Assert(t, data.FreqTfs["tall"][0] == float64(0.3333333333333333))
	Assert(t, data.FreqTfs["tall"][1] == float64(0.5))
	Assert(t, data.FreqTfs["tall"][2] == float64(1))

	// Check for term frequency's per 'document' (blonde)
	Assert(t, data.FreqTfs["blonde"][0] == float64(0.5))

}

func TestTfIdClassifier_ConvertToTfIdf(t *testing.T) {
	c := NewClassifierTfIdf(Good, Bad)
	Assert(t, c.IsTfIdf() == true)

	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"tall", "blonde"}, Good)
	c.Learn([]string{"tall"}, Good)

	// Now we convert the TF's to Tf/Idf
	// We can only this after we have learned all the documents and classes.
	// We can add more learning afterwards but need to call ConvertToTfIdf() again before
	// we can predict classes.
	c.ConvertTermsFreqToTfIdf()

	data := c.datas[Good]

	// Tf-Idf after we have converted the tf's
	Assert(t, data.Freqs["tall"] == float64(0.5620939930012151))
	Assert(t, data.Freqs["blonde"] == float64(0.16440195389316542))
	Assert(t, data.Freqs["notseen"] == float64(0))
	Assert(t, data.FreqTfs["tall"][0] == float64(0.11664504260744213))
	Assert(t, data.FreqTfs["tall"][1] == float64(0.16440195389316542))
	Assert(t, data.FreqTfs["tall"][2] == float64(0.28104699650060755))

}

func TestTfIdClassifier_CheckForDoubleConvert(t *testing.T) {

	c := NewClassifierTfIdf(Good, Bad)
	Assert(t, c.IsTfIdf() == true)

	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"tall", "blonde"}, Good)
	c.Learn([]string{"tall"}, Good)

	// We can only call ConverToTdfIdf once per learning cycle (cumulative counts).
	c.ConvertTermsFreqToTfIdf()

	defer func() {
		if err := recover(); err != nil {
			// we are good
		}
	}()
	c.ConvertTermsFreqToTfIdf()
	Assert(t, false, "Should have panicked:Can only run ConvertTermsFreqToTfIdf() once after a learning cycle.", c)

}

func TestTfIdClassifier_LogScore(t *testing.T) {
	c := NewClassifierTfIdf(Good, Bad)
	Assert(t, c.IsTfIdf() == true)

	c.Learn([]string{"tall", "handsome", "rich"}, Good)
	c.Learn([]string{"tall", "blonde"}, Good)
	c.Learn([]string{"tall"}, Good)
	c.Learn([]string{"fat"}, Bad)
	c.Learn([]string{"short", "poor"}, Bad)

	c.ConvertTermsFreqToTfIdf()

	score, likely, strict := c.LogScores([]string{"the", "tall", "man"})

	Assert(t, score[0] == float64(-53.028113582945196))
	Assert(t, score[0] > score[1], "Class 'Good' should be closer to 0 than Class 'Bad' - both will be negative") // this is good
	Assert(t, likely == 0, "Class should be 'Good'")
	Assert(t, strict == true, "No tie's")
	fmt.Printf("%#v", score)

}
