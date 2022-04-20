# Send and receive messages

```
kubectl -n kafka run kafka-producer -ti --image=quay.io/strimzi/kafka:0.28.0-kafka-3.1.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic my-topic
```

Open other terminal:

```
kubectl -n kafka run kafka-consumer -ti --image=quay.io/strimzi/kafka:0.28.0-kafka-3.1.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic my-topic --from-beginning
```

## Benchmark

## hardware

3 nodes
6 Core Intel Xenon 2.5GHz
6*7200RPM SATA
32G RAM
1Gb Ethernet

no RAID JBOD style

## no comsumer,  all messages are persisted but not read

821,557 records/sec
78.3 MB/sec

## Single producer thread, 3x asynchronous replication

786,980 records/sec
75.1 MB/sec

## Single producer thread, 3x synchronous replication

421,823 records/sec
40.2 MB/sec

## Three producers, 3x async replication

2,024,032 records/sec
193.0 MB/sec

we perform just as well after writing a TB of data, as we do for the first few hundred MBs

## Single Consumer

940,521 records/sec
89.7 MB/sec

## Three Consumers

2,615,968 records/sec
249.5 MB/sec

## Producer and Consumer

795,064 records/sec
75.8 MB/sec

## Effect of Message Size

the raw count of records we can send per second decreases as the records get bigger. But if we look at MB/second, we see that the total byte throughput of real user data increases as messages get bigger

## End-to-end Latency

2 ms (median)
3 ms (99th percentile)
14 ms (99.9th percentile)
