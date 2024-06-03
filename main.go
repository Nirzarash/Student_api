package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Student struct {
	EnrollmentNumber string `json:"enrollmentNumber"`
	Name             string `json:"name"`
	Age              int    `json:"age"`
	Class            string `json:"class"`
	Subject          string `json:"subject"`
	IsDeleted        bool   `json:"isDeleted"`
}

var (
	students = make(map[string]Student)
	mu       sync.Mutex
	logger   *log.Logger
)

func main() {
	logFile, err := os.OpenFile("student.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	logger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	router := mux.NewRouter()
	router.HandleFunc("/student/v1/students", createStudent).Methods("POST")
	router.HandleFunc("/student/v1/students", getAllStudents).Methods("GET")
	router.HandleFunc("/student/v1/students/{studentId}", getStudent).Methods("GET")
	router.HandleFunc("/student/v1/students/{studentId}", deleteStudent).Methods("DELETE")

	logger.Println("Server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	err := json.NewDecoder(r.Body).Decode(&student)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Println("Error decoding request body:", err)
		return
	}

	student.EnrollmentNumber = uuid.New().String()
	mu.Lock()
	students[student.EnrollmentNumber] = student
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"enrollmentNumber": student.EnrollmentNumber})
	logger.Println("Created student with enrollment number:", student.EnrollmentNumber)
}

func getAllStudents(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var result []Student
	for _, student := range students {
		if !student.IsDeleted {
			result = append(result, student)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	logger.Println("Fetched all students")
}

func getStudent(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	studentId := params["studentId"]

	mu.Lock()
	student, exists := students[studentId]
	mu.Unlock()

	if !exists || student.IsDeleted {
		http.Error(w, "Student not found", http.StatusNotFound)
		logger.Println("Student not found with enrollment number:", studentId)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
	logger.Println("Fetched student with enrollment number:", studentId)
}

func deleteStudent(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	studentId := params["studentId"]

	mu.Lock()
	student, exists := students[studentId]
	if exists && !student.IsDeleted {
		student.IsDeleted = true
		students[studentId] = student
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		logger.Println("Soft deleted student with enrollment number:", studentId)
		return
	}
	mu.Unlock()

	http.Error(w, "Student not found", http.StatusNotFound)
	logger.Println("Student not found with enrollment number:", studentId)
}
