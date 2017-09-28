package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/boltdb/bolt"
)

const (
	indexHTML = `<html>
<body>
<form action="/login" method="POST">
<label>UserID: <input name="userid"></label>
<label>Password: <input name="password" type="password"></label>
<input type="submit" value="login">
</form>
<a href="/register">if you've not have a account, register</a>
</body>
</html>`
	registerHTML = `<html>
<body>
<form action="/register" method="POST">
<label>UserID: <input name="userid"></label>
<label>Password: <input name="password" type="password"></label>
<input type="submit" value="register">
</form>
</body>
</html>`
	userPageHTML = `<html>
<body>
username: %s, password: %s<br />
<a href="/update">update information</a>
</body>
</html>`
	updateHTML = `<html>
<body>
<form action="/update" method="POST">
<label>Password: <input name="password" type="password"></label>
<input type="submit" value="register">

<input type="hidden" name="csrf-token" value="%s">
</form>
</body>
</html>`
)

func main() { os.Exit(exec()) }

func exec() int {
	// initialize database
	if err := initialize(); err != nil {
		log.Printf("%s", err)
		return 1
	}

	// bind handlers
	http.HandleFunc(`/`, indexHandler)
	http.HandleFunc(`/register`, registerHandler)
	http.HandleFunc(`/login`, loginHandler)
	http.HandleFunc(`/userpage`, userPageHandler)
	http.HandleFunc(`/update`, updateHandler)

	// run HTTP server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("%s", err)
		return 1
	}
	return 0
}

func initialize() error {
	db, err := bolt.Open("bolt.db", 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.CreateBucketIfNotExists([]byte(`users`))
	if err != nil {
		return err
	}
	return nil
}

func errorResponse(w http.ResponseWriter, statusCode int, err error) {
	w.WriteHeader(statusCode)
	w.Write([]byte(err.Error()))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(indexHTML))
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Write([]byte(registerHTML))
	case "POST":
		userid := r.FormValue(`userid`)
		password := r.FormValue(`password`)
		if userid == "" || password == "" {
			errorResponse(w, http.StatusInternalServerError, errors.New(`userid or password is nil`))
			return
		}
		db, err := bolt.Open("bolt.db", 0600, nil)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err)
			return
		}
		defer db.Close()
		db.Update(func(tx *bolt.Tx) error {
			b, _ := tx.CreateBucketIfNotExists([]byte(`users`))
			if before := b.Get([]byte(userid)); before != nil {
				return errors.New(`user id has used`)
			}
			return b.Put([]byte(userid), []byte(password))
		})
		w.Header().Set(`Location`, `/`)
		w.WriteHeader(http.StatusFound)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case `POST`:
		userid := r.FormValue(`userid`)
		givenPass := r.FormValue(`password`)
		db, err := bolt.Open("bolt.db", 0600, nil)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err)
			return
		}
		defer db.Close()
		var password []byte
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(`users`))
			password = b.Get([]byte(userid))
			if password == nil {
				return errors.New(`no user found`)
			}
			return nil
		})
		if string(password) != givenPass {
			w.Header().Set(`Location`, `/login`)
			w.WriteHeader(http.StatusFound)
			return
		}
		c := http.Cookie{
			Name:  "sessionid",
			Value: userid, // oops.
		}
		http.SetCookie(w, &c)
		w.Header().Set(`Location`, `/userpage`)
		w.WriteHeader(http.StatusFound)
	}
}

func session(r *http.Request) (string, bool) {
	db, err := bolt.Open("bolt.db", 0600, nil)
	if err != nil {
		return "", false
	}
	defer db.Close()
	c, err := r.Cookie(`sessionid`)
	if err != nil {
		return "", false
	}
	var password []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(`users`))
		password = b.Get([]byte(c.Value))
		return nil
	})
	if password == nil {
		return "", false
	}
	return string(password), true
}

func userPageHandler(w http.ResponseWriter, r *http.Request) {
	password, ok := session(r)
	if !ok {
		errorResponse(w, http.StatusForbidden, errors.New(`not authorized`))
		return
	}
	c, err := r.Cookie(`sessionid`)
	if err != nil {
		w.Header().Set(`Location`, `/`)
		w.WriteHeader(http.StatusFound)
		return
	}
	w.Write([]byte(fmt.Sprintf(userPageHTML, c.Value, password)))
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	csrfToken := `someRandomToken`
	switch r.Method {
	case `GET`:
		_, ok := session(r)
		if !ok {
			return
		}
		w.Write([]byte(fmt.Sprintf(updateHTML, "someRandomToken")))
		return
	case `POST`:
		tk := r.FormValue(`csrf-token`)
		if tk != csrfToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		db, err := bolt.Open("bolt.db", 0600, nil)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, err)
			return
		}
		defer db.Close()
		db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(`users`))
			c, err := r.Cookie(`sessionid`)
			if err != nil {
				return err
			}
			return b.Put([]byte(c.Value), []byte(r.FormValue(`password`)))
		})
		w.Header().Set(`Location`, `/userpage`)
		w.WriteHeader(http.StatusFound)
		return
	}
}
