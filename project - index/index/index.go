package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func handleClient(conn net.Conn) {
	defer conn.Close()
	clientAddress := conn.LocalAddr().(*net.TCPAddr).IP.String()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println(err)
		return
	}

	message := strings.TrimSpace(string(buf[:n]))
	parts := strings.Split(message, ":")
	if len(parts) < 3 {
		conn.Write([]byte("Invalid command\n"))
		return
	}

	command := parts[0]
	data := parts[1]
	port := parts[2]

	switch command {
	case "register_client":
		clientAddress = clientAddress + ":" + port
		_, err = db.Exec("INSERT INTO clients (address) VALUES (?)", clientAddress)

		if err != nil {
			log.Println(err)
			conn.Write([]byte("Error registering client\n"))
			return
		}
		fmt.Println("Client enregistré :", clientAddress)
		conn.Write([]byte("Client enregistré avec succès\n"))

	case "register_file":
		var clientID int
		err = db.QueryRow("SELECT id FROM clients WHERE address = ?", clientAddress+":"+port).Scan(&clientID)

		if err != nil {
			log.Println(err)
			conn.Write([]byte("Error finding client\n"))
			return
		}
		_, err = db.Exec("INSERT INTO files (client_id, file_name) VALUES (?, ?)", clientID, data)
		if err != nil {
			log.Println(err)
			conn.Write([]byte("Error registering file\n"))
			return
		}
		fmt.Println("File enregistré :", data)
		conn.Write([]byte("File enregistré avec succès\n"))

	case "search_file":
		var clientAddress string
		err = db.QueryRow(`
            SELECT clients.address 
            FROM files 
            JOIN clients ON files.client_id = clients.id
            WHERE files.file_name = ?`, data).Scan(&clientAddress)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Println("File not found:", data)
				conn.Write([]byte("File not found\n"))
			} else {
				log.Println("Error executing search query:", err)
				conn.Write([]byte("Error searching for file\n"))
			}
			return
		}
		conn.Write([]byte(fmt.Sprintf("File found, owner IP and Port:%s\n", clientAddress)))

	default:
		conn.Write([]byte("Unknown command\n"))
	}
}

func main() {

	// Initialise la base de données SQLite
	var err error
	db, err = sql.Open("sqlite3", "../clients.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Crée la table clients si elle n'existe pas déjà
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS clients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		address TEXT UNIQUE
	);`)
	if err != nil {
		log.Fatalf("Error creating clients table: %v", err)
	}
	// Crée la table files si elle n'existe pas déjà
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id INTEGER,
		file_name TEXT,
		FOREIGN KEY(client_id) REFERENCES clients(id)
	);`)

	if err != nil {
		log.Fatal(err)
	}
	// Démarre le serveur TCP sur le port 8082
	listener, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Serveur TCP démarré, en attente de connexions...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go handleClient(conn)
	}
}
