package bootloader

import (
	"log"
	"testing"
)

func Test_a(t *testing.T) {
	p := newProperties("prop-")
	type db struct {
		Username string
		Password string
	}
	type Config struct {
		Port int
		db   db
	}

	type User struct {
		Name     string
		Password string
	}
	c := &Config{}
	c.Port = 80
	c.db.Username = "root"
	c.db.Password = "toor"

	u := &User{}
	u.Name = "admin"
	u.Password = "admin123"
	users := make(map[string]interface{})
	users["user"] = u

	p.set(c)
	p.set(users)

	log.Println(p.data)

	if int(p.value("port").Int()) != c.Port {
		t.Errorf("port not found")
	}

	if p.value("db.username").String() != c.db.Username {
		t.Errorf("db.Username not found")
	}

	if p.value("db.password").String() != c.db.Password {
		t.Errorf("db.password not found")
	}

}
