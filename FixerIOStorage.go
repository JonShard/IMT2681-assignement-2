package main

import (
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type FixerIOStorage struct {
	DatabaseURL    string
	DatabaseName   string
	CollectionName string
}

func (fios *FixerIOStorage) Update(url string) error {

	// delete old entry
	session, err := mgo.Dial(fios.DatabaseURL)
	defer session.Close()
	if err != nil {
		return err
	}

	err = session.DB(fios.DatabaseName).DropDatabase()
	if err != nil {
		return err
	}

	// UPDATE LATEST

	// get payload
	payload, err := fetchFixerIO(url+"/latest?base=EUR", getJSON)
	if err != nil {
		return err
	}

	// fetch info we want
	rates := payload.Rates
	rates["EUR"] = 1

	mRates := MongoRate{Name: "latest", Rates: rates}

	// store it
	session, err = mgo.Dial(fios.DatabaseURL)
	if err != nil {
		return err
	}
	defer session.Close()

	err = session.DB(fios.DatabaseName).C(fios.CollectionName).Insert(mRates)
	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
		return err
	}

	// UPDATE AVERAGE
	average, err := generateAverage(url)
	if err != nil {
		return err
	}

	mRates = MongoRate{Name: "average", Rates: average}

	// store it
	session, err = mgo.Dial(fios.DatabaseURL)
	if err != nil {
		return err
	}
	defer session.Close()

	err = session.DB(fios.DatabaseName).C(fios.CollectionName).Insert(mRates)
	if err != nil {
		fmt.Printf("error in Insert(): %v", err.Error())
		return err
	}

	return nil
}

func generateAverage(url string) (map[string]float32, error) {

	payload, err := fetchFixerIO(url+"/latest?base=EUR", getJSON)
	if err != nil {
		return nil, err
	}
	temp := payload.Rates

	t := time.Now()

	// get for last 7 days
	for i := 0; i < 6; i++ {
		t = t.AddDate(0, 0, -1)
		date := t.Format("2006-01-02")

		payload, err := fetchFixerIO(url+"/"+date+"?base=EUR", getJSON)
		if err != nil {
			return nil, err
		}

		rateForDay := payload.Rates
		for k, v := range rateForDay {
			temp[k] += v
		}
	}

	// do average

	for k := range temp {
		temp[k] /= 7
	}

	// ..add EUR (=1)
	temp["EUR"] = 1

	return temp, nil
}

func (fios *FixerIOStorage) getRate(curr1, curr2, name string) (float32, error) {

	// fetch rates from mongodb
	session, err := mgo.Dial(fios.DatabaseURL)
	if err != nil {
		return 923, err
	}
	defer session.Close()

	var mrate MongoRate
	err = session.DB(fios.DatabaseName).C(fios.CollectionName).Find(bson.M{"name": name}).One(&mrate)
	if err != nil {
		return 923, err
	}

	// calculate rate
	rate1, ok1 := mrate.Rates[curr1]
	rate2, ok2 := mrate.Rates[curr2]
	if !ok1 || !ok2 {
		return 923, errInvalidCurrency
	}

	return rate2 / rate1, nil
}

func (fios *FixerIOStorage) Latest(curr1, curr2 string) (float32, error) {

	rate, err := fios.getRate(curr1, curr2, "latest")
	if err != nil {
		return 923, err
	}

	return rate, nil
}

func (fios *FixerIOStorage) Average(curr1, curr2 string) (float32, error) {

	rate, err := fios.getRate(curr1, curr2, "average")
	if err != nil {
		return 923, err
	}

	return rate, nil
}

type MongoRate struct {
	Name  string             `name`
	Rates map[string]float32 `rates`
}
