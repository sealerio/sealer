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

package imagedistributor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	sealerConfig "github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/pkg/infradriver"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	lp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type p2pDistributor struct {
	ipfsNode         icore.CoreAPI
	ipfsAPI          *core.IpfsNode
	ipfsCancel       context.CancelFunc
	ipfsContext      context.Context
	sshInfraDriver   infradriver.InfraDriver
	imageMountInfo   []ClusterImageMountInfo
	registryCacheDir string
	rootfsCacheDir   string
	configs          []v1.Config
	options          DistributeOption
}

func (p *p2pDistributor) DistributeRegistry(deployHosts []net.IP, dataDir string) error {
	for _, info := range p.imageMountInfo {
		if !osi.IsFileExist(filepath.Join(info.MountDir, RegistryDirName)) {
			continue
		}

		logrus.Infof("Distributing %s", info.MountDir)

		if err := p.distributeImpl(deployHosts, info.MountDir, dataDir); err != nil {
			return fmt.Errorf("failed to distribute: %s", err)
		}
	}

	return nil
}

func (p *p2pDistributor) Distribute(hosts []net.IP, dest string) error {
	for _, info := range p.imageMountInfo {
		logrus.Infof("Distributing %s", info.MountDir)
		if err := p.dumpConfigToRootfs(info.MountDir); err != nil {
			return err
		}

		if err := p.renderRootfs(info.MountDir); err != nil {
			return err
		}

		if err := p.distributeImpl(hosts, info.MountDir, dest); err != nil {
			return fmt.Errorf("failed to distribute: %s", err)
		}
	}

	return nil
}

func (p *p2pDistributor) Restore(targetDir string, hosts []net.IP) error {
	if !p.options.Prune {
		return nil
	}

	rmRootfsCMD := fmt.Sprintf("rm -rf %s", targetDir)

	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range hosts {
		host := ip
		eg.Go(func() error {
			err := p.sshInfraDriver.CmdAsync(host, nil, rmRootfsCMD)
			if err != nil {
				return fmt.Errorf("faild to delete rootfs on host [%s]: %v", host.String(), err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (p *p2pDistributor) distributeImpl(deployHosts []net.IP, dir string, dest string) error {
	name, err := tarGzDirectory(dir)
	if err != nil {
		return err
	}
	logrus.Infof("Compressed %s", dir)

	ipfsDirectory, err := getUnixfsNode(name)
	if err != nil {
		return fmt.Errorf("failed to prepare rootfs %s: %s", dir, err)
	}

	cid, err := p.ipfsNode.Unixfs().Add(p.ipfsContext, ipfsDirectory, func(uas *options.UnixfsAddSettings) error {
		uas.Chunker = "size-1048576"
		uas.FsCache = true
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to add rootfs %s to IPFS network: %s", dir, err)
	}
	logrus.Infof("Loaded %s", dir)

	cidString := cidToString(cid)

	eg, _ := errgroup.WithContext(context.Background())

	for _, ip := range deployHosts {
		host := ip

		eg.Go(func() error {
			logrus.Infof("%s start", host)

			localIP, err := p.getRealIP(host)
			if err != nil {
				return fmt.Errorf("failed to distribute to host %s: %s", host, err)
			}

			localID := p.ipfsAPI.Identity.String()
			localHost := fmt.Sprintf("/ip4/%s/tcp/40011/p2p/%s", localIP, localID)

			command := fmt.Sprintf("/usr/bin/dist-receiver -bootstrap %s -cid %s -filename %s -target %s", localHost, cidString, name, dest)
			if _, err = p.sshInfraDriver.Cmd(host, nil, command); err != nil {
				return fmt.Errorf("failed to distribute to host %s: %s", host, err)
			}

			logrus.Infof("%s done. ", host)

			return nil
		})
	}

	if err := waitForState(deployHosts, "1"); err != nil {
		return err
	}

	remote := ""
	known, err := p.ipfsNode.Swarm().KnownAddrs(p.ipfsContext)
	if err != nil {
		return err
	}

	for _, host := range known {
		remote = remote + fmt.Sprintf("%s,", host)
	}

	encoded := b64.StdEncoding.EncodeToString([]byte(remote))

	for _, ip := range deployHosts {
		host := ip
		go func() {
			_, _ = http.Get(fmt.Sprintf("http://%s:4002/connect?target=%s", host, encoded))
			_, _ = http.Get(fmt.Sprintf("http://%s:4002/next", host))
		}()
	}

	if err := waitForState(deployHosts, "2"); err != nil {
		return err
	}

	goNext(deployHosts)

	if err := eg.Wait(); err != nil {
		return err
	}

	if err := os.Remove(name); err != nil {
		logrus.Warnf("Failed to delete intermediate file %s: %s", name, err)
	}

	return nil
}

func (p *p2pDistributor) getRealIP(host net.IP) (string, error) {
	localIPBytes, err := p.sshInfraDriver.Cmd(host, nil, "echo $SSH_CLIENT | awk '{print $1}'")
	if err != nil {
		return "", fmt.Errorf("failed to distribute to host %s: %s", host, err)
	}

	localIP := string(localIPBytes[:])
	localIP = strings.TrimSpace(localIP)

	return localIP, nil
}

func cidToString(cid path.Resolved) string {
	fullCid := cid.String()
	cidParts := strings.Split(fullCid, "/")
	var cidString string
	for _, s := range cidParts {
		cidString = s
	}
	return cidString
}

func NewP2PDistributor(
	imageMountInfo []ClusterImageMountInfo,
	driver infradriver.InfraDriver,
	configs []v1.Config,
	options DistributeOption,
) (Distributor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	node, api, err := spawnNode(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create IPFS node: %s", err)
	}

	return &p2pDistributor{
		ipfsNode:         node,
		ipfsAPI:          api,
		ipfsCancel:       cancel,
		ipfsContext:      ctx,
		sshInfraDriver:   driver,
		imageMountInfo:   imageMountInfo,
		registryCacheDir: filepath.Join(driver.GetClusterRootfsPath(), "cache", RegistryCacheDirName),
		rootfsCacheDir:   filepath.Join(driver.GetClusterRootfsPath(), "cache", RootfsCacheDirName),
		configs:          configs,
		options:          options,
	}, nil
}

func spawnNode(ctx context.Context) (icore.CoreAPI, *core.IpfsNode, error) {
	// Create a Temporary Repo
	repoPath, err := createTempRepo()
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

func createTempRepo() (string, error) {
	err := setupPlugins("")
	if err != nil {
		return "", err
	}

	repoPath, err := os.MkdirTemp("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return "", err
	}

	var bs []string
	cfg.Bootstrap = bs

	var listen []string
	listen = append(listen, "/ip4/0.0.0.0/tcp/40011")
	cfg.Addresses.Swarm = listen

	// Create the repo with the config
	if err := fsrepo.Init(repoPath, cfg); err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
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
		Host: constructPeerHost,
		Repo: repo,
	}

	return core.NewNode(ctx, nodeOptions)
}

func constructPeerHost(id peer.ID, ps peerstore.Peerstore, options ...lp2p.Option) (host.Host, error) {
	pkey := ps.PrivKey(id)
	if pkey == nil {
		return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
	}
	options = append([]lp2p.Option{lp2p.Identity(pkey), lp2p.Peerstore(ps)}, options...)
	return lp2p.New(options...)
}

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error injecting plugins: %s", err)
	}

	return nil
}

func (p *p2pDistributor) dumpConfigToRootfs(mountDir string) error {
	return sealerConfig.NewConfiguration(mountDir).Dump(p.configs)
}

func (p *p2pDistributor) renderRootfs(mountDir string) error {
	var (
		renderEtc       = filepath.Join(mountDir, "etc")
		renderChart     = filepath.Join(mountDir, "charts")
		renderManifests = filepath.Join(mountDir, "manifests")
		renderData      = p.sshInfraDriver.GetClusterEnv()
	)

	for _, dir := range []string{renderEtc, renderChart, renderManifests} {
		if osi.IsFileExist(dir) {
			err := env.RenderTemplate(dir, renderData)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func tarGzDirectory(sourceDir string) (string, error) {
	h := sha256.New()
	h.Write([]byte(sourceDir))
	name := hex.EncodeToString(h.Sum(nil))

	filename := fmt.Sprintf("%s.tar.gz", name)

	if err := createTarGz(sourceDir, filename); err != nil {
		return "", fmt.Errorf("failed to uncompress resource file: %s", err)
	}

	return filename, nil
}

func waitForState(hosts []net.IP, expectedStage string) error {
	for {
		good := 0
		eg, _ := errgroup.WithContext(context.Background())

		for _, ip := range hosts {
			host := ip

			eg.Go(func() error {
				resp, err := http.Get(fmt.Sprintf("http://%s:4002/stage", host))
				if err != nil {
					return err
				}

				if contentBytes, err := io.ReadAll(resp.Body); err != nil {
					return err
				} else if string(contentBytes) == expectedStage {
					good++
				}

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			continue
		}

		if good == len(hosts) {
			return nil
		}

		time.Sleep(time.Second)
	}
}

func goNext(hosts []net.IP) {
	for _, ip := range hosts {
		host := ip
		go func() {
			_, _ = http.Get(fmt.Sprintf("http://%s:4002/next", host))
		}()
	}
}

func createTarGz(sourceDir, filename string) error {
	tarFile, err := os.Create(filepath.Clean(filename))
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzWriter := gzip.NewWriter(tarFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filePath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(filepath.Clean(filePath))
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})
}
