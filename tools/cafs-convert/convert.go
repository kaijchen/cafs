package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kaijchen/cafs/config"
	"github.com/kaijchen/cafs/metadata"
)

func sha256sum(path string) (checksum string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func stashTo(pool string, zpool string) func(path string) string {
	if pool == "" {
		return sha256sum
	}
	return func(path string) (checksum string) {
		checksum = sha256sum(path)
		caspath := filepath.Join(pool, checksum)
		if _, err := os.Stat(caspath); os.IsNotExist(err) {
			os.Link(path, filepath.Join(pool, checksum))
		}
		zpath := filepath.Join(zpool, checksum+".zst")
		if _, err := os.Stat(zpath); os.IsNotExist(err) {
			exec.Command("zstd", "-o", zpath, path).Run()
		}
		return
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %v root meta [pool]\n", os.Args[0])
		os.Exit(-1)
	}
	root, meta := os.Args[1], os.Args[2]
	var pool, zpool string
	if len(os.Args) < 4 {
		if cfg, err := config.GetDefaultConfig(); err == nil {
			pool = cfg.Pool
			zpool = cfg.Zpool
		}
	} else {
		pool = os.Args[3]
	}
	tree := metadata.Tree{}
	if err := tree.Build(root, stashTo(pool, zpool)); err != nil {
		fmt.Println(err)
	}
	tree.Save(meta)
}
