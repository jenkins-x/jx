## Naive Bayesian Classification

Perform naive Bayesian classification into an arbitrary number of classes on sets of strings. `bayesian` also supports term frequency-inverse document frequency calculations ([TF-IDF](https://www.wikiwand.com/en/Tf%E2%80%93idf)).

Copyright (c) 2011-2017. Jake Brukhman. (jbrukh@gmail.com).
All rights reserved.  See the LICENSE file for BSD-style license.

------------

### Background

This is meant to be an low-entry barrier Go library for basic Bayesian classification. See code comments for a refresher on naive Bayesian classifiers, and please take some time to understand underflow edge cases as this otherwise may result in innacurate classifications.

------------

### Installation

Using the go command:
```shell
go get github.com/jbrukh/bayesian
go install !$
```
------------

### Documentation

See the GoPkgDoc documentation [here](https://godoc.org/github.com/jbrukh/bayesian).

------------

### Features

- Conditional probability and "log-likelihood"-like scoring.
- Underflow detection.
- Simple persistence of classifiers.
- Statistics.
- TF-IDF support.

------------

### Example 1 (Simple Classification)

To use the classifier, first you must create some classes
and train it:

```go
import . "bayesian"

const (
    Good Class = "Good"
    Bad Class = "Bad"
)

classifier := NewClassifier(Good, Bad)
goodStuff := []string{"tall", "rich", "handsome"}
badStuff  := []string{"poor", "smelly", "ugly"}
classifier.Learn(goodStuff, Good)
classifier.Learn(badStuff,  Bad)
```
Then you can ascertain the scores of each class and
the most likely class your data belongs to:
```go
scores, likely, _ := classifier.LogScores(
                        []string{"tall", "girl"}
                     )
```
Magnitude of the score indicates likelihood. Alternatively (but
with some risk of float underflow), you can obtain actual probabilities:

```go
probs, likely, _ := classifier.ProbScores(
                        []string{"tall", "girl"}
                     )
```

### Example 2 (TF-IDF Support)

To use the TF-IDF classifier, first you must create some classes
and train it and you need to call ConvertTermsFreqToTfIdf() AFTER training
and before calling classification methods such as `LogScores`, `SafeProbScores`, and `ProbScores`)

```go
import . "bayesian"

const (
    Good Class = "Good"
    Bad Class = "Bad"
)

// Create a classifier with TF-IDF support.
classifier := NewClassifierTfIdf(Good, Bad)

goodStuff := []string{"tall", "rich", "handsome"}
badStuff  := []string{"poor", "smelly", "ugly"}

classifier.Learn(goodStuff, Good)
classifier.Learn(badStuff,  Bad)

// Required
classifier.ConvertTermsFreqToTfIdf()
```

Then you can ascertain the scores of each class and
the most likely class your data belongs to:

```go
scores, likely, _ := classifier.LogScores(
                        []string{"tall", "girl"}
                     )
```
Magnitude of the score indicates likelihood. Alternatively (but
with some risk of float underflow), you can obtain actual probabilities:

```go
probs, likely, _ := classifier.ProbScores(
                        []string{"tall", "girl"}
                     )
```
Use wisely.
