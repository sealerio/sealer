// Copyright © 2023 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	b64 "encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var stageNow int
var sig chan struct{}
var targets []string

func main() {
	bootstrap := flag.String("bootstrap", "", "Specify the bootstrap node")
	cidArg := flag.String("cid", "", "Specify the CID")
	fileName := flag.String("filename", "", "Specify the name of the file to be distributed")
	targetDir := flag.String("target", "", "Specify the target directory")

	flag.Parse()

	stageNow = 0
	sig = make(chan struct{})

	server := &http.Server{
		Addr:         "0.0.0.0:4002",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  600 * time.Second,
	}

	http.HandleFunc("/stage", stage)
	http.HandleFunc("/next", next)
	http.HandleFunc("/connect", connect)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(fmt.Errorf("failed to spawn command receiver: %s", err))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		if err := os.RemoveAll(*fileName); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to clean up %s: %s\n", *fileName, err)
		}
	}()

	node, _, err := spawnEphemeral(ctx, *bootstrap)
	if err != nil {
		panic(fmt.Errorf("failed to spawn ephemeral node: %s", err))
	}

	if err := connectToPeers(ctx, node, []string{*bootstrap}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to root peer: %s\n", err)
	}

	stageNow = 1

	<-sig

	if err := connectToPeers(ctx, node, targets); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to peers: %s\n", err)
	}

	cid := icorepath.New(*cidArg)

	if err := node.Dht().Provide(ctx, cid); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed the resource file: %s\n", err)
	}

	if err := node.Pin().Add(ctx, cid); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to pin the resource file: %s\n", err)
	}

	rootNode, err := node.Unixfs().Get(ctx, cid)

	if err != nil {
		panic(fmt.Errorf("could not get file with CID: %s", err))
	}

	if err := files.WriteTo(rootNode, *fileName); err != nil {
		panic(fmt.Errorf("could not write out the fetched CID: %s", err))
	}

	if err := os.MkdirAll(*targetDir, os.ModePerm); err != nil {
		panic(fmt.Errorf("failed to create target directory: %s", err))
	}

	if err := extractTarGz(*fileName, *targetDir); err != nil {
		panic(fmt.Errorf("failed to uncompress resource file: %s", err))
	}

	stageNow = 2

	<-sig
}

func createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	return core.NewNode(ctx, nodeOptions)
}

func spawnEphemeral(ctx context.Context, bootstrap string) (icore.CoreAPI, *core.IpfsNode, error) {
	if err := setupPlugins(""); err != nil {
		return nil, nil, err
	}

	repoPath, err := createTempRepo(bootstrap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	node, err := createNode(ctx, repoPath)
	if err != nil {
		return nil, nil, err
	}

	api, err := coreapi.NewCoreAPI(node)

	return api, node, err
}

func setupPlugins(externalPluginsPath string) error {
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error injecting plugins: %s", err)
	}

	return nil
}

func createTempRepo(bootstrap string) (string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return "", err
	}

	cfg.Bootstrap = append(cfg.Bootstrap, bootstrap)

	if err := fsrepo.Init(repoPath, cfg); err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				panic(fmt.Errorf("failed to connect to %s: %s", peerInfo.ID, err))
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func stage(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "%d", stageNow)
}

func next(w http.ResponseWriter, req *http.Request) {
	sig <- struct{}{}
}

func connect(w http.ResponseWriter, req *http.Request) {
	target := req.URL.Query().Get("target")
	targetDecoded, err := b64.StdEncoding.DecodeString(target)
	if err != nil {
		return
	}

	target = string(targetDecoded)

	targets = strings.Split(target, ",")
}

func extractTarGz(src, dest string) error {
	tarFile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzipReader, err := gzip.NewReader(tarFile)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// we created the tarball ourselves so it is safe
		// #nosec G305
		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// it is expected to create directory with permission 755
			// #nosec G301
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(filepath.Clean(target))
			if err != nil {
				return err
			}
			defer outFile.Close()

			// we created the tarball ourselves so it is safe
			// #nosec G110
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
		}
	}

	return nil
}
