csrf-example
============

## requirement

* [boltdb/bolt](https://github.com/boltdb/bolt)

## howto

### CSRF attack
* `$ go run server/main.go`
* access to `localhost:8080`
* register some user (this is an example, **DO NOT INPUT YOUR PASSWORD USED FOR OTHER SITE**) and login
* check your registered user info at `localhost:8080/userpage`
* open evil/evil.html
* refresh `localhost:8080/userpage`

### Protect from CSRF attack
* checkout `secure` branch
* check diff from `master`
* one more do CSRF attack from access
