package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Product struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Price        float64   `json:"price"`
	Availability bool      `json:"availability"`
	Category     string    `json:"category"`
	CreatedAt    time.Time `json:"created_at"`
}

type ProductCatalog struct {
	Products []Product `json:"products"`
}

var ProductList = []Product{
	{ID: 1, Name: "Product 1", Price: 10.50, Availability: true, Category: "Premium", CreatedAt: time.Now()},
	{ID: 2, Name: "Product 2", Price: 5.50, Availability: false, Category: "Regular", CreatedAt: time.Now()},
	{ID: 3, Name: "Product 3", Price: 2.50, Availability: true, Category: "Budget", CreatedAt: time.Now()},
	{ID: 4, Name: "Product 4", Price: 12.50, Availability: true, Category: "Premium", CreatedAt: time.Now()},
	{ID: 5, Name: "Product 5", Price: 7.50, Availability: false, Category: "Regular", CreatedAt: time.Now()},
	{ID: 6, Name: "Product 6", Price: 9.50, Availability: false, Category: "Premium", CreatedAt: time.Now()},
}

var products = map[int]Product{}

func init() {
	for _, product := range ProductList {
		products[product.ID] = product
	}
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/products", getProductCatalogHandler).Methods(http.MethodGet)
	r.HandleFunc("/products/{id}", getProductHandler).Methods(http.MethodGet)

	fmt.Println("Product Service is listening on :8000...")
	err := http.ListenAndServe(":8000", r)
	if err != nil {
		panic(err)
	}
}

func getProductCatalogHandler(w http.ResponseWriter, r *http.Request) {
	productCatalog := ProductCatalog{Products: ProductList}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(productCatalog)
}

func getProductHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productIDStr := vars["id"]

	productID, err := strconv.Atoi(productIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid product ID: %s", productIDStr)
		return
	}

	product, ok := products[productID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Product not found: %d", productID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
