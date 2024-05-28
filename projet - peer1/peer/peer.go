package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	PORT := "8081"
	go func() {
		defer wg.Done()
		client(PORT)
	}()

	go func() {
		defer wg.Done()
		server(PORT)
	}()

	wg.Wait()
}

func client(PORT string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("Choose an option: [join | publish | search]")
		scanner.Scan()
		option := scanner.Text()

		switch option {
		case "join":
			registerClient(PORT)
		case "publish":
			publishFile(PORT)
		case "search":
			searchFile()
		default:
			fmt.Println("Invalid option. Please choose again.")
		}
	}
}

func registerClient(PORT string) {
	conn, err := net.Dial("tcp", "localhost:8082")
	if err != nil {
		fmt.Println("Error connecting to the server:", err)
		return
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "register_client:%s:%s\n", "", PORT)
	if err != nil {
		fmt.Println("Error sending join request:", err)
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return
	}

	fmt.Println("Server response:", response)
}

func publishFile(PORT string) {
	fmt.Println("Enter the file name to publish:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	fileName := scanner.Text()
	dir := "../peer1/"
	// Check if the file exists in the specified directory
	if _, err := os.Stat(dir + fileName + ".txt"); os.IsNotExist(err) {
		fmt.Println("File does not exist.")
		return
	}

	conn, err := net.Dial("tcp", "localhost:8082")
	if err != nil {
		fmt.Println("Error connecting to the server:", err)
		return
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "register_file:%s:%s\n", fileName, PORT)
	if err != nil {
		fmt.Println("Error sending publish request:", err)
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return
	}

	fmt.Println("Server response:", response)
}

func searchFile() {
	fmt.Println("Enter the file name to search:")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	fileName := scanner.Text()

	conn, err := net.Dial("tcp", "localhost:8082")
	if err != nil {
		fmt.Println("Error connecting to the server:", err)
		return
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "search_file:%s:%s\n", fileName, "")
	if err != nil {
		fmt.Println("Error sending search request:", err)
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println("Error reading server response:", err)
		return
	}
	if strings.HasPrefix(response, "File found, owner IP and Port:") {
		peerAddress := strings.TrimPrefix(response, "File found, owner IP and Port:")
		fmt.Println("File found at:", peerAddress)
		requestFileContent(peerAddress, fileName)
	} else {
		fmt.Println("File not found.")
	}
}

func requestFileContent(peerAddress, fileName string) {
	// Split the peerAddress into address and port
	add := strings.Split(peerAddress, ":")
	portp := add[len(add)-1] // Get the last element of the array
	portp = strings.TrimSpace(portp)
	// Dial the connection using the separated address and port
	conn, err := net.Dial("tcp", "localhost:"+portp)

	if err != nil {
		fmt.Println("Error connecting to the peer:", err)
		return
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "send:%s\n", fileName)
	if err != nil {
		fmt.Println("Error sending file request:", err)
		return
	}

	file, err := os.Create("../peer1/" + fileName + ".txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, conn)
	if err != nil {
		fmt.Println("Error receiving file:", err)
		return
	}

	fmt.Println("File received successfully.")
}

func server(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Peer server is listening on port " + port + "...")
	for { // for each peer a new connection
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handlePeerRequest(conn)
	}
}

func handlePeerRequest(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println(err)
		return
	}

	message := strings.TrimSpace(string(buf[:n]))
	parts := strings.Split(message, ":")

	if len(parts) < 2 {
		fmt.Println("Invalid request format")
		return
	}

	command := parts[0]
	fileName := parts[1]

	switch command {
	case "send":
		sendFileContent(conn, fileName)
	default:
		fmt.Println("Unknown command")
	}
}

func sendFileContent(conn net.Conn, fileName string) {
	file, err := os.Open("../peer1/" + fileName + ".txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		fmt.Fprintf(conn, "Error: file not found\n")
		return
	}
	defer file.Close()

	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println("Error sending file:", err)
	}
}
