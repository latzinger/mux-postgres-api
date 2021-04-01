package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/latzinger/mux-postgres-api/model"
)

var app Application

const (
	createTableQuery = `CREATE TABLE IF NOT EXISTS products
(
    id SERIAL,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT products_pkey PRIMARY KEY (id)
)`
)

func TestMain(m *testing.M) {
	app.Init(
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"))

	checkTableExists()
	exitCode := m.Run()
	clearTable()
	os.Exit(exitCode)
}

// Helpe Functions

func checkTableExists() {
	if _, err := app.DB.Exec(createTableQuery); err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	app.DB.Exec("DELETE FROM products")
	app.DB.Exec("ALTER SEQUENCE products_id_seq RESTART WITH 1")
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	app.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if actual != expected {
		t.Errorf("Expected response code is %d. Got %d", expected, actual)
	}
}

func addProducts(count int) {

	if count < 1 {
		count = 1
	}

	for i := 0; i < count; i++ {
		app.DB.Exec("INSERT INTO products(name, price) VALUES($1, $2)", "Product "+strconv.Itoa(i), (i+1.0)*10)
	}

}

// Tests

func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/products", nil)
	res := executeRequest(req)

	checkResponseCode(t, http.StatusOK, res.Code)

	bodyBytes, _ := ioutil.ReadAll(res.Body)

	if body := string(bodyBytes); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}

}

func TestGetNonExistentProduct(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/product/99", nil)
	res := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, res.Code)

	var m map[string]string
	json.Unmarshal(res.Body.Bytes(), &m)

	if m["error"] != "Product not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Product not found'. Got '%s'", m["error"])
	}

}

func TestCreateProduct(t *testing.T) {
	clearTable()

	p := model.Product{
		Name:  "test product",
		Price: 11.22,
	}

	jsonString, _ := json.Marshal(p)
	req, _ := http.NewRequest("POST", "/product", bytes.NewBuffer(jsonString))
	req.Header.Set("Content-Type", "application/json")
	res := executeRequest(req)

	checkResponseCode(t, http.StatusCreated, res.Code)

	json.Unmarshal(res.Body.Bytes(), &p)

	if p.Name != "test product" {
		t.Errorf("Expected product name to be 'test product'. Got '%v'", p.Name)
	}

	if p.Price != 11.22 {
		t.Errorf("Expected product price to be '11.22'. Got '%v'", p.Price)
	}

	if p.ID != 1 {
		t.Errorf("Expected product ID to be '1'. Got '%v'", p.ID)
	}

}

func TestGetProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	res := executeRequest(req)

	checkResponseCode(t, http.StatusOK, res.Code)

}

func TestUpdateProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	res := executeRequest(req)

	originalProduct := model.Product{}
	json.Unmarshal(res.Body.Bytes(), &originalProduct)

	updatedProduct := model.Product{
		Name:  "test product - updated name",
		Price: 11.22,
	}

	jsonString, _ := json.Marshal(updatedProduct)
	req, _ = http.NewRequest("PUT", "/product/1", bytes.NewBuffer(jsonString))
	req.Header.Set("Content-Type", "application/json")
	res = executeRequest(req)

	checkResponseCode(t, http.StatusOK, res.Code)

	p := model.Product{}
	json.Unmarshal(res.Body.Bytes(), &p)

	if p.ID != originalProduct.ID {
		t.Errorf("Expected the id to remain the same (%v). Got %v", originalProduct.ID, p.ID)
	}

	if p.Name == originalProduct.Name {
		t.Errorf("Expected the name to change from '%v' to '%v'. Got '%v'", originalProduct.Name, updatedProduct.Name, p.Name)
	}

	if p.Price == originalProduct.Price {
		t.Errorf("Expected the price to change from '%v' to '%v'. Got '%v'", originalProduct.Price, updatedProduct.Price, p.Price)
	}

}

func TestDeleteProduct(t *testing.T) {
	clearTable()
	addProducts(1)

	req, _ := http.NewRequest("GET", "/product/1", nil)
	res := executeRequest(req)
	checkResponseCode(t, http.StatusOK, res.Code)

	req, _ = http.NewRequest("DELETE", "/product/1", nil)
	res = executeRequest(req)
	checkResponseCode(t, http.StatusOK, res.Code)

	req, _ = http.NewRequest("GET", "/product/1", nil)
	res = executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, res.Code)
}
