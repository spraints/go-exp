package main

import (
  "fmt"
  "io"
  "log"
  "os"
  "os/exec"
  "time"
)

type Writer struct {
  out chan<- []byte
}

func (w *Writer) WriteLines(lines ...string) {
  for _, line := range lines {
    io.WriteString(w, line + "\n")
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

type TFS struct {
  repositoryServiceUrl string
  username string
  password string
}

func (tfs *TFS) FetchChangesets(changesets chan<- *Changeset) {
  start := 1
  max := 256
  for {
    maxChangeset := tfs.QueryHistory("$/CRST", start, max, changesets)
    if maxChangeset == -1 {
      break;
    } else {
      start = maxChangeset + 1
    }
  }
}

func (tfs *TFS) QueryHistory(path string, start, max int, changesets chan<- *Changeset) int {
  requestXml := "<itemSpec item=\"" + path + "\" recurse=\"Full\" did=\"0\"/>"
  requestXml += "<versionItem xsi:type=\"LatestVersionSpec\"/>"
  requestXml += fmt.Sprintf("<versionFrom xsi:type=\"ChangesetVersionSpec\" cs=\"%d\"/>", start)
  requestXml += fmt.Sprintf("<maxCount>%d</maxCount>", max)
  requestXml += "<includeFiles>true</includeFiles>"
  requestXml += "<generateDownloadUrls>true</generateDownloadUrls>"
  requestXml += "<slotMode>true</slotMode>"
  requestXml += "<sortAscending>true</sortAscending>"
  responseXml := tfs.Soap("QueryHistory", requestXml)
  fmt.Println(responseXml)
  return -1
}

func (tfs *TFS) Soap(action, xml string) string {
  xml = "<?xml version=\"1.0\"?>" +
    "<soap:Envelope xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:soap=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns=\"http://schemas.microsoft.com/TeamFoundation/2005/06/VersionControl/ClientServices/03\">" +
    "<soap:Body>" +
    "<" + action + ">" + xml + "</" + action + ">" +
    "</soap:Body>" +
    "</soap:Envelope>"
  // todo
  // maybe this will help? https://gist.github.com/bemasher/1224702
  return xml
}

func FetchChangesets(changesets chan<- *Changeset) {
  tfs := new(TFS)
  tfs.repositoryServiceUrl = "https://tfs.codeplex.com/tfs/TFS04/Services/1.0/repository.asmx"
  tfs.username = os.Getenv("TFS_USERNAME")
  tfs.password = os.Getenv("TFS_PASSWORD")
  tfs.FetchChangesets(changesets)
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
    for _, change := range changeset.Changes {
      change.Apply(writer)
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
