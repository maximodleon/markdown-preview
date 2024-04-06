package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

type content struct {
  Title string
  Body template.HTML
}

const (
  defaultTemplate = `<!DOCTYPE html>
 <html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=utf-8">
    <title>{{ .Title }}</title>
  </head>
  <body>
{{ .Body }}
  </body>
</html>
`
)

func main() {
  filename := flag.String("file", "", "Markdown file to preview")
  skipPreview := flag.Bool("s", false, "Skip auto preview")
  tFname := flag.String("t", "", "Alternate template name")
  flag.Parse()

  if *filename == "" {
    flag.Usage()
    os.Exit(1)
  }

  if err := run(*filename, *tFname, os.Stdout, *skipPreview); err != nil {
     fmt.Fprintln(os.Stderr, err)
     os.Exit(1)
  }
}

func run(filename string, tFname string, out io.Writer, skipPreview bool) error {

  input, err := os.ReadFile(filename)
  if err != nil {
    return err
  }

  htmlData, err := parseContent(input, tFname)

  if err != nil {
     return err
  }

  temp, err := os.CreateTemp("", "mdp*.html")

  if err != nil {
    return err
  }

  if err := temp.Close(); err != nil {
    return err
  }

  outName := temp.Name()
  fmt.Fprintln(out, outName)

  if err := saveHTML(outName, htmlData); err != nil {
    return err
  }

  if skipPreview {
    return nil
  }

  // Delete file after function returns
  defer os.Remove(outName)

  return preview(outName)
}

func parseContent(input []byte, tFname string) ([]byte, error) {
  // Pase the markdown file
  // To generate a valid HTML
  output := blackfriday.Run(input)
  body := bluemonday.UGCPolicy().SanitizeBytes(output)

  t, err := template.New("mdp").Parse(defaultTemplate)

  if err != nil {
     return nil, err
  }

  // Use default template
  // if no file is provided
  if tFname != "" {
    t, err = template.ParseFiles(tFname)

    if err != nil {
      return nil, err
    }
  }

  // Intantiate the conten type
  c := content{
    Title: "Markdown Preview Tool",
    Body: template.HTML(body),
  }

  var buffer bytes.Buffer

  if err := t.Execute(&buffer, c); err != nil {
    return nil, err
  }

  return buffer.Bytes(), nil
}

func saveHTML(outName string, data []byte) error {
  return os.WriteFile(outName, data, 0644)
}

func preview(fname string) error {
  cName := ""
  cParams := []string{}

  switch runtime.GOOS {
    case "linux":
      cName = "xdg-open"
    case "windows":
      cName = "cmd.exe"
      cParams = []string{"/C", "start"}
    case "darwin":
      cName = "open"
    default:
      return fmt.Errorf("OS not supported")
  }

  // Append filename to parameters slice
  cParams = append(cParams, fname)
  // Locate executable
  cPath, err := exec.LookPath(cName)

  if err != nil {
    return err
  }

  // Open the file using the command
  err = exec.Command(cPath, cParams...).Run()

  // Delay so the file can be opened
  // before it is deleted
  time.Sleep(2 * time.Second)
  return err
}
