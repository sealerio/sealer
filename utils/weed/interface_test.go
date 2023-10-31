// Copyright © 2021 Alibaba Group Holding Ltd.
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

package weed

// This should run with root permission.

//func clean() {
//}
//
//func TestDeployer_CreateEtcdCluster1(t *testing.T) {
//	err := NewDeployer(&Config{
//		BinDir:         "./test/bin3",
//		DataDir:        "./test/data3",
//		LogDir:         "./test/log3",
//		MasterIP:       []string{"127.0.0.1:1111", "127.0.0.1:2222", "127.0.0.1:3333"},
//		VolumeIP:       []string{"127.0.0.1:4444", "127.0.0.1:5555", "127.0.0.1:6666"},
//		PidDir:         "./test/pid3",
//		CurrentIP:      "127.0.0.1",
//		PeerPort:       3333,
//		ClientPort:     2390,
//		EtcdConfigPath: "./test/etcd3.conf",
//	}).CreateEtcdCluster(context.Background())
//	if err != nil {
//		t.Error(err)
//		return
//	}
//}

//
//func TestDeployer_CreateEtcdCluster2(t *testing.T) {
//	err := NewDeployer(&Config{
//		BinDir:         "./test/bin1",
//		DataDir:        "./test/data1",
//		LogDir:         "./test/log1",
//		MasterIP:       []string{"127.0.0.1:1111", "127.0.0.1:2222", "127.0.0.1:3333"},
//		PidDir:         "./test/pid1",
//		CurrentIP:      "127.0.0.1",
//		PeerPort:       1111,
//		ClientPort:     2391,
//		EtcdConfigPath: "./test/etcd1.conf",
//	}).CreateEtcdCluster(context.Background())
//	if err != nil {
//		t.Error(err)
//		return
//	}
//}
//
//func TestDeployer_CreateEtcdCluster3(t *testing.T) {
//	err := NewDeployer(&Config{
//		BinDir:         "./test/bin2",
//		DataDir:        "./test/data2",
//		LogDir:         "./test/log2",
//		MasterIP:       []string{"127.0.0.1:1111", "127.0.0.1:2222", "127.0.0.1:3333"},
//		PidDir:         "./test/pid2",
//		CurrentIP:      "127.0.0.1",
//		PeerPort:       2222,
//		ClientPort:     2392,
//		EtcdConfigPath: "./test/etcd2.conf",
//	}).CreateEtcdCluster(context.Background())
//	if err != nil {
//		t.Error(err)
//		return
//	}
//}
//
//func TestDownloadWeed(t *testing.T) {
//	d := &deployer{}
//	err := d.downloadWeed()
//	assert.Nil(t, err)
//}
