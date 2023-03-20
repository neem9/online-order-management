package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

const (
	productServiceHost = "http://localhost:8000/"
)

type Product struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Price          float64   `json:"price"`
	InventoryCount int       `json:"inventory_count"`
	Category       string    `json:"category"`
	CreatedAt      time.Time `json:"created_at"`
}

type ProductCatalog struct {
	Products []Product `json:"products"`
}

type OrderItem struct {
	ProductID    int     `json:"product_id"`
	ProductPrice float64 `json:"product_price"`
	ProductQty   int     `json:"product_qty"`
}

type Order struct {
	ID               int         `json:"id"`
	Items            []OrderItem `json:"items"`
	Value            float64     `json:"value"`
	Status           string      `json:"status"`
	Discount         float64     `json:"discount"`
	DispatchDate     time.Time   `json:"dispatch_date"`
	CreationDateTime time.Time   `json:"creation_date_time"`
}

var (
	orders  = make(map[int]*Order)
	orderID = 1
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/orders", getOrdersHandler).Methods(http.MethodGet)
	r.HandleFunc("/orders", createOrderHandler).Methods(http.MethodPost)
	r.HandleFunc("/orders/{id}", updateOrderHandler).Methods(http.MethodPatch)

	fmt.Println("Order Server is listening on :8001...")
	err := http.ListenAndServe(":8001", r)
	if err != nil {
		panic(err)
	}
}

func getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ordersList := make([]*Order, 0, len(orders))
	for _, order := range orders {
		ordersList = append(ordersList, order)
	}
	json.NewEncoder(w).Encode(ordersList)
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Decoding the request
	var order Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	// Get product catalogue from the product service
	productCatalog, err := getProductCatalog()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to get product catalog: %v", err)
		return
	}

	productsMap := make(map[int]*Product)
	for i := range productCatalog.Products {
		productsMap[productCatalog.Products[i].ID] = &productCatalog.Products[i]
	}

	premiumProductCount := 0
	orderValue := float64(0)
	for _, orderItem := range order.Items {
		product := productsMap[orderItem.ProductID]

		// Check if enough products are available in the inventory
		if product.InventoryCount < orderItem.ProductQty {
			w.WriteHeader(http.StatusConflict)
			fmt.Fprintf(w, "There is not enough of %q to fulfill this order", product.Name)
			return
		}

		// Calculate order value
		orderValue += product.Price * float64(orderItem.ProductQty)
		orderItem.ProductPrice = product.Price

		// Count the premium products in the order
		if product.Category == "Premium" {
			premiumProductCount++
		}
	}

	// Calculate the discault and order value
	order.Value = orderValue
	if premiumProductCount >= 3 {
		order.Discount = 10
		order.Value -= order.Value * order.Discount / 100
	}

	// Create the order
	order.ID = orderID
	order.CreationDateTime = time.Now()
	order.Status = "Placed"

	// Update the catalogue
	err = updateProductCatalog(productsMap, &order)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to update product catalog: %v", err)
		return
	}

	// Save the order
	orders[order.ID] = &order
	orderID++
	json.NewEncoder(w).Encode(order)
}

func updateOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid order ID: %s", idStr)
		return
	}

	order, ok := orders[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Order not found: %d", id)
		return
	}

	var update Order
	err = json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to parse request body: %v", err)
		return
	}

	if update.Status != "" {
		if update.Status != "Dispatched" && update.Status != "Completed" && update.Status != "Cancelled" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid order status: %s", update.Status)
			return
		}
		order.Status = update.Status
	}

	if order.Status == "Dispatched" {
		if update.DispatchDate.IsZero() {
			update.DispatchDate = time.Now()
		}
		order.DispatchDate = update.DispatchDate
	}

	// TODO: If order status is cancelled, return the product back to the inventory

	orders[id] = order

	json.NewEncoder(w).Encode(order)
}

func getProductCatalog() (*ProductCatalog, error) {
	resp, err := http.Get(productServiceHost + "/products")
	if err != nil {
		return nil, err
	}

	productCatalog := &ProductCatalog{}
	err = json.NewDecoder(resp.Body).Decode(productCatalog)
	if err != nil {
		return nil, err
	}

	return productCatalog, nil
}

func updateProductCatalog(products map[int]*Product, order *Order) error {
	client := &http.Client{}
	url := productServiceHost + "/products"

	productList := make([]Product, 0, len(order.Items))
	for _, orderItem := range order.Items {
		product := *products[orderItem.ProductID]

		product.InventoryCount -= orderItem.ProductQty
		if product.InventoryCount < 0 {
			return fmt.Errorf("invalid inventory count")
		}

		productList = append(productList, product)
	}

	payload, err := json.Marshal(productList)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create update product request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do update product request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update product request failed with status: %s", resp.Status)
	}

	return nil
}
