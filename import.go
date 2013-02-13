package main

import (
  "fmt"
  "io"
  "log"
  "os/exec"
  "time"
)

type Writer struct {
  out chan<- []byte
}

func (w *Writer) WriteLines(lines ...string) {
  for i := range lines {
    io.WriteString(w, lines[i] + "\n")
  }
}

func (w *Writer) Write(data []byte) (int, error) {
  w.out <- data
  return len(data), nil
}


type Author struct {
  Name string
  Email string
}

type Change interface {
  Apply(writer *Writer)
}

type Changeset struct {
  Author Author
  Date time.Time
  Message string
  Changes []Change
}

func FetchChangesets(changesets chan<- *Changeset) {
  changeset := new(Changeset)
  changeset.Author.Name = "Matt Burke"
  changeset.Author.Email = "spraints@gmail.com"
  changeset.Date = time.Now()
  changeset.Message = "Hi!"
  changesets <- changeset
  close(changesets)
}

func ConvertChangesets(changesets <-chan *Changeset, importData chan<- []byte) {
  writer := new(Writer)
  writer.out = importData
  for changeset := range changesets {
    writer.WriteLines("commit refs/heads/import",
      fmt.Sprintf("committer %s <%s> %d -0000", changeset.Author.Name, changeset.Author.Email, changeset.Date.UTC().Unix()),
      "data <<END_OF_COMMIT_MESSAGE_FI",
      changeset.Message,
      "END_OF_COMMIT_MESSAGE_FI")
    for i := range changeset.Changes {
      changeset.Changes[i].Apply(writer)
    }
    writer.WriteLines("")
  }
  close(importData)
}

func FastImport(importData <-chan []byte, complete chan<- bool) {
  // Assume that 'target' is already initialized
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
  complete <- true
}

func main() {
  importData := make(chan []byte)
  changesets := make(chan *Changeset)
  done := make(chan bool)

  go FastImport(importData, done)
  go ConvertChangesets(changesets, importData)
  go FetchChangesets(changesets)
  <-done
}
