package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sjwhitworth/golearn/base"
	"github.com/sjwhitworth/golearn/evaluation"
	"github.com/sjwhitworth/golearn/knn"
)

func main() {
	// Открываем CSV файл
	f, err := os.Open("output01.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Читаем CSV файл
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	rawData := base.NewDenseInstances()
	x := &base.FloatAttribute{"x", 14}
	y := &base.FloatAttribute{"y", 14}
	goalX := &base.FloatAttribute{"goalX", 0}
	goalY := &base.FloatAttribute{"goalY", 0}
	cost := &base.FloatAttribute{"cost", 0}

	xSpec := rawData.AddAttribute(x)
	ySpec := rawData.AddAttribute(y)
	goalXSpec := rawData.AddAttribute(goalX)
	goalYSpec := rawData.AddAttribute(goalY)
	costSpec := rawData.AddAttribute(cost)
	rawData.AddClassAttribute(cost)

	rawData.Extend(len(records))
	for i, v := range records[1:] {
		rawData.Set(xSpec, i, xSpec.GetAttribute().GetSysValFromString(strings.TrimSpace(v[1])))
		rawData.Set(ySpec, i, ySpec.GetAttribute().GetSysValFromString(strings.TrimSpace(v[2])))
		rawData.Set(goalXSpec, i, goalXSpec.GetAttribute().GetSysValFromString(strings.TrimSpace(v[4])))
		rawData.Set(goalYSpec, i, goalYSpec.GetAttribute().GetSysValFromString(strings.TrimSpace(v[5])))
		rawData.Set(costSpec, i, costSpec.GetAttribute().GetSysValFromString(strings.TrimSpace(v[3])))
	}

	// Print a pleasant summary of your data.
	fmt.Println(rawData)

	//Initialises a new KNN classifier
	cls := knn.NewKnnClassifier("euclidean", "linear", 2)

	//Do a training-test split
	trainData, testData := base.InstancesTrainTestSplit(rawData, 0.80)
	cls.Fit(trainData)
	//Calculates the Euclidean distance and returns the most popular label
	predictions, err := cls.Predict(testData)
	if err != nil {
		panic(err)
	}

	// Prints precision/recall metrics
	confusionMat, err := evaluation.GetConfusionMatrix(testData, predictions)
	if err != nil {
		panic(fmt.Sprintf("Unable to get confusion matrix: %s", err.Error()))
	}
	fmt.Println(evaluation.GetSummary(confusionMat))
}
