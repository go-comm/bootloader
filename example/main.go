package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-comm/bootloader"
)

type User struct {
	Username string
	Role     string
}

type UserService struct {
	User        *User
	RoleService *RoleService `bloader:"auto"`
}

func (s *UserService) OnCreate() {
	s.User = &User{
		Username: "root",
	}
}

func (s *UserService) GetUser() *User {
	s.User.Role = s.RoleService.GetRole()
	log.Printf("%+v", s.User)
	return s.User
}

type RoleService struct {
	Role string
}

func (s *RoleService) OnCreate() {
	s.Role = "admin"
}

func (s *RoleService) GetRole() string {
	return s.Role
}

type Server struct {
	UserService *UserService `bloader:"user-service"`
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.UserService.GetUser())
}

func (s *Server) OnStart() {
	http.HandleFunc("/", s.home)
	log.Println("server started.")
	http.ListenAndServe(":8888", nil)
}

func main() {

	bootloader.AddModulerFromType(new(RoleService))
	bootloader.AddModuler("user-service", new(UserService))
	bootloader.AddModulerFromType(new(Server))

	bootloader.Launch()
}
