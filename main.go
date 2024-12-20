package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Address представляет адрес клиента.
type Address struct {
	City   string `json:"city"`
	Street string `json:"street"`
}

// Client представляет клиента.
type Client struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Age          int       `json:"age"`
	RegisterDate time.Time `json:"registerDate"`
	FavCoffee    string    `json:"favCoffee"`
	Address      Address   `json:"address"`
}

// Welcome используется для отображения приветственной страницы.
type Welcome struct {
	Name string
	Time string
}

var (
	clients   = make(map[int]Client) // Хранилище клиентов
	clientsMu sync.Mutex             // Мьютекс для защиты данных клиентов
)

func main() {
	// Динамическое приветствие
	welcome := Welcome{"Гость", time.Now().Format(time.Stamp)}
	templates := template.Must(template.ParseFiles("templates/main.html"))

	// Эндпоинт для статики
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Главная страница
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if name := r.FormValue("name"); name != "" {
			welcome.Name = name
		}
		if err := templates.ExecuteTemplate(w, "main.html", welcome); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Эндпоинты для работы с клиентами
	http.HandleFunc("/addClient", addClientHandler)
	http.HandleFunc("/deleteClient", deleteClientHandler)
	http.HandleFunc("/getClients", getClientsHandler)

	// Настройка сервера
	srv := &http.Server{
		Addr: ":8090",
	}

	go func() {
		fmt.Println("Сервер запущен на http://localhost:8090")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Ошибка сервера: %v\n", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Ошибка остановки сервера: %+v\n", err)
	}
	fmt.Println("Сервер остановлен")
}

// addClientHandler добавляет клиента.
func addClientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Неверный метод запроса", http.StatusMethodNotAllowed)
		return
	}

	var newClient Client
	if err := json.NewDecoder(r.Body).Decode(&newClient); err != nil {
		http.Error(w, "Ошибка парсинга тела запроса", http.StatusBadRequest)
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if _, exists := clients[newClient.ID]; exists {
		http.Error(w, "Клиент с таким ID уже существует", http.StatusConflict)
		return
	}

	clients[newClient.ID] = newClient
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newClient)
}

// deleteClientHandler удаляет клиента.
func deleteClientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Неверный метод запроса", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || idStr == "" {
		http.Error(w, "Неверный или отсутствующий ID", http.StatusBadRequest)
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if _, exists := clients[id]; !exists {
		http.Error(w, "Клиент не найден", http.StatusNotFound)
		return
	}

	delete(clients, id)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Клиент с ID %d успешно удален", id)
}

// getClientsHandler возвращает всех клиентов.
func getClientsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Неверный метод запроса", http.StatusMethodNotAllowed)
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}
