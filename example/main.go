package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"

	"github.com/go-comm/bootloader"
)

type User struct {
	Username string
}
type RuntimeInfo struct {
	GOOS    string
	Version string
	NumCPU  int
}

func FuncRuntimeInfo() (interface{}, error) {

	return &RuntimeInfo{
		GOOS:    runtime.GOOS,
		Version: runtime.Version(),
		NumCPU:  runtime.NumCPU(),
	}, nil
}

type UserService struct {
	User *User
}

func (s *UserService) OnCreate() {
	s.User = &User{
		Username: "root",
	}
}

func (s *UserService) GetUser() *User {
	log.Printf("%+v", s.User)
	return s.User
}

type Server struct {
	UserService *UserService `bloader:"user-service"`
	RuntimeInfo *RuntimeInfo `bloader:"auto"`
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.UserService.GetUser())
	json.NewEncoder(w).Encode(s.RuntimeInfo)
}

func (s *Server) OnStart() {
	http.HandleFunc("/", s.home)
	log.Println("server started.")
	http.ListenAndServe(":8888", nil)
}

func main() {
	bootloader.AddFromType(FuncRuntimeInfo)
	bootloader.Add("user-service", new(UserService))
	bootloader.AddFromType(new(Server))

	bootloader.Launch()
}
