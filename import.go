package main

import (
  "net/http"
  "os/exec"
  "time"
  "log"
)

type Author struct {
  name string
  email string
}

type Change interface {
  Apply(importData chan string)
}

type Changeset struct {
  author Author
  date time.Time
  changes []Change
}

func FastImport(importData <-chan []byte) {
  cmd := exec.Command("git", "--git-dir=target", "fast-import")
  stdin, err := cmd.StdinPipe()
  if err != nil {
    log.Fatal(err)
  }
  err = cmd.Start()
  if err != nil {
    log.Fatal(err)
  }
  for data := range importData {
    stdin.Write(data)
  }
}

func main() {
  importData := make(chan []byte)
  changesets := make(chan *Changeset)

  go FastImport(importstdin)
  go ChangesetConverter(changesets, importstdin)
  go ChangesetPump(changesets)
}
