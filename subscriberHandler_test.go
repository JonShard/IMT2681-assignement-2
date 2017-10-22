package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// make request with given method+body, assert statuscode and return pointer
//	to response (nil if failure)
func reqTest(t *testing.T, ts *httptest.Server, target, method string, body io.Reader,
	expectedCode int, msg string) *http.Response {

	// instantiate test client
	client := &http.Client{}

	// create a request to our mock HTTP server
	url := ts.URL + target
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Errorf("error constructing valid request (%s)", msg)
		return nil
	}

	// do request
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("error doing valid request (%s). Error: %s", msg, err.Error())
		return nil
	}

	if resp == nil {
		return nil
	}

	// reach to the response from the request
	if resp.StatusCode != expectedCode {
		t.Errorf("expected statuscode %d, received %d. (%s)", expectedCode,
			resp.StatusCode, msg)
	}

	return resp
}

// TODO: add test case for malformed URL :D
func TestSubscriberHandler_handleSubscriberRequest_POST(t *testing.T) {

	// instantiate test handler using volatile db (shouldn't fail)
	db := VolatileSubscriberDBFactory()
	handler := SubscriberHandlerFactory(&db)

	// instantiate mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(handler.handleSubscriberRequest))
	defer ts.Close()

	// fully valid
	validBody := strings.NewReader(`{
		"webhookURL": "http://remoteUrl:8080/randomWebhookPath",
		"baseCurrency": "EUR",
		"targetCurrency": "NOK",
		"minTriggerValue": 1.50, 
		"maxTriggerValue": 2.55
		}`)

	// json correct, but missing one field: invalid (TODO: doesn't work)
	invalidBody := strings.NewReader(`{
		"webhookURL": "http://remoteUrl:8080/randomWebhookPath",
		"baseCurrency": "EUR",
		"targetCurrency": "NOK",
		"maxTriggerValue": 2.55
		}`)

	// json incorrect, invalid
	veryInvalidBody := strings.NewReader(`{
		"webhookURL": "http://remoteUrl:8080/randomWebhookPath"",
		}`)

	// asssert that correct error codes are returned (store valid response)
	reqTest(t, ts, "", http.MethodPost, invalidBody, http.StatusBadRequest,
		"POST invalid json: malformed")
	reqTest(t, ts, "", http.MethodPost, veryInvalidBody, http.StatusBadRequest,
		"POST invalid json: missing field")
	resp := reqTest(t, ts, "", http.MethodPost, validBody, http.StatusOK,
		"POST valid json")

	// test valid response:

	if resp == nil {
		t.Error("erroring in getting response from server")
	}

	// assert that body is a single int
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("error converting body in response to []byte")
	}

	str := string(b)
	_, err = strconv.Atoi(str)
	if err != nil {
		t.Error("body in response to subscriber request isn't an id (int)")
	}
}

func TestSubscriberHandler_handleSubscriberRequest_GET(t *testing.T) {

	// instantiate test handler using volatile db (shouldn't fail)
	db := VolatileSubscriberDBFactory()
	handler := SubscriberHandlerFactory(&db)

	// instantiate mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(handler.handleSubscriberRequest))
	defer ts.Close()

	// test ids
	validID := 1
	invalidID := 2

	// sneak stuff into the db
	url := "testing"
	testSub := Subscriber{WebhookURL: &url}
	db.subscribers[validID] = testSub

	// assert that request for valid id returns OK
	resp := reqTest(t, ts, "/"+strconv.Itoa(validID), http.MethodGet, http.NoBody, http.StatusOK,
		"GET valid id")

	// assert that request for invalid id doesn't succeed
	reqTest(t, ts, "/"+strconv.Itoa(invalidID), http.MethodGet, http.NoBody, http.StatusNotFound,
		"GET invalid id")

	// assert that malformed request returns bad request
	reqTest(t, ts, "/THISISNOTANIDxD", http.MethodGet, http.NoBody, http.StatusBadRequest,
		"GET malformed id")

	// test body of response from valid request:

	if resp == nil {
		t.Error("error getting response from server")
		return
	}

	// attempt to unmarshall
	var s Subscriber
	err := json.NewDecoder(resp.Body).Decode(&s)
	if err != nil {
		t.Error("error while unmarshalling response:", err.Error())
		return
	}

	// assert that it contains our test data
	if *s.WebhookURL != *testSub.WebhookURL {
		t.Error("returned wrong subscriber from get request")
	}

}
