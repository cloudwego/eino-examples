package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

type LocalOperator struct{}

func (l *LocalOperator) ReadFile(ctx context.Context, path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return err.Error(), nil
	}
	return string(b), nil
}

func (l *LocalOperator) WriteFile(ctx context.Context, path string, content string) error {
	return os.WriteFile(path, []byte(content), 0666)
}

func (l *LocalOperator) IsDirectory(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (l *LocalOperator) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (l *LocalOperator) RunCommand(ctx context.Context, command string) (string, error) {
	cs := strings.Fields(command)
	cmd := exec.Command(cs[0], cs[1:]...)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		return errBuf.String(), nil
	}
	return outBuf.String(), nil
}
