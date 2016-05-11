package main

import (
    "golang.org/x/crypto/scrypt"
    "fmt"
)


func main() {
    dk, err := scrypt.Key([]byte("some password"), []byte("salt"), 16384, 8, 1, 32)
    if err != nil {
        panic(err)
    }
    fmt.Println("dk=", dk)
    fmt.Println(string(dk), ", len=", len(dk))
}

/*import (
    "golang.org/x/crypto/bcrypt"
    "fmt"
)

func main() {
    password := []byte("MyDarkSecret")
    fmt.Println("pwd=", string(password))

    // Hashing the password with the default cost of 10
    hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
    if err != nil {
        panic(err)
    }
    fmt.Println("hashed=", string(hashedPassword), ", len=", len(hashedPassword))

    // Comparing the password with the hash
    err = bcrypt.CompareHashAndPassword(hashedPassword, password)
    fmt.Println(err) // nil means it is a match
}*/