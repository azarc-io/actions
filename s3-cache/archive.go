package main

import (
	"archive/zip"
	"fmt"
	"github.com/klauspost/pgzip"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Zip - Create .zip file and add dirs and files that match glob patterns
func Zip(filename string, artifacts []string) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	archive := zip.NewWriter(outFile)
	defer archive.Close()

	for _, pattern := range artifacts {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}

		for _, match := range matches {
			filepath.Walk(match, func(path string, info os.FileInfo, err error) error {
				header, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}

				header.Name = path
				header.Method = zip.Deflate

				writter, err := archive.CreateHeader(header)
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()

				_, err = io.Copy(writter, file)

				return err
			})
		}
	}

	return nil
}

// Unzip - Unzip all files and directories inside .zip file
func Unzip(filename string) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := os.MkdirAll(filepath.Dir(file.Name), os.ModePerm); err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			continue
		}

		outFile, err := os.OpenFile(file.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		currentFile, err := file.Open()
		if err != nil {
			return err
		}

		if _, err = io.Copy(outFile, currentFile); err != nil {
			return err
		}

		outFile.Close()
		currentFile.Close()
	}

	return nil
}

func ZipParallel(filename string, artifacts []string) error {
	var (
		start = time.Now()
		queue = make(chan string, 100)
		eg    = errgroup.Group{}
	)

	target, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	writer, err := pgzip.NewWriterLevel(target, pgzip.DefaultCompression)

	for _, pattern := range artifacts {
		p := pattern
		eg.Go(func() error {
			matches, err := filepath.Glob(p)
			if err != nil {
				return err
			}

			for _, match := range matches {
				if err := filepath.Walk(match, func(path string, info os.FileInfo, err error) error {
					if info.IsDir() {
						return nil
					}
					queue <- path
					return err
				}); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		panic(err)
	} else {
		close(queue)
	}

	go func() {
		for i := 0; i < 3; i++ {
			for path := range queue {
				if err := addToStream(writer, path); err != nil {
					panic(err)
				}
			}
		}
	}()

	elapsed := time.Since(start)
	log.Println(fmt.Sprintf("finished compressing in: %s", elapsed))

	return nil
}

func UnzipParallel(filename string) error {
	//reader, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0600)
	//if err != nil {
	//	return err
	//}
	//defer reader.Close()
	//r, err := pgzip.NewReader(reader)
	//if err != nil {
	//	return err
	//}

	return nil
}

func addToStream(writer *pgzip.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return nil
}
