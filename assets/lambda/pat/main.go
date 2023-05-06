package main

import (
    "fmt"
    "math/rand"
)

const patPrefix = "animal_pat_"
const patLength = int(64)

var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randSeq(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

func main() {
    fmt.Printf("%s%s", patPrefix, randSeq(patLength))
}

