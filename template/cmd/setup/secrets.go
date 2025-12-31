package main

import (
    "io"
    "strings"
    "fmt"
)

func handleToken(r io.Reader, w io.Writer) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if strings.TrimSpace(cfg.PHPSESSID) != "" {
		update, err := promptYesNo(r, w, "PHPSESSID found. Update it?", false)
		if err != nil {
			return err
		}
		if !update {
			return nil
		}
	}

	fmt.Fprint(w, "Enter PHPSESSID (leave blank to skip): ")
	line, err := readLine(r)
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	cfg.PHPSESSID = line
	return saveConfig(cfg)
}