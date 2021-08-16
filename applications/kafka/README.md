# USAGE
Kafka can be accessed by consumers via port 9092 on the following DNS name from within your cluster:

    my-kafka.kafka-system.svc.cluster.local

Each Kafka broker can be accessed by producers via port 9092 on the following DNS name(s) from within your cluster:

    my-kafka-0.my-kafka-headless.kafka-system.svc.cluster.local:9092

To create a pod that you can use as a Kafka client run the following commands:

    kubectl run my-kafka-client --restart='Never' --image docker.io/bitnami/kafka:2.8.0-debian-10-r61 --namespace kafka-system --command -- sleep infinity
    kubectl exec --tty -i my-kafka-client --namespace kafka-system -- bash

    PRODUCER:
        kafka-console-producer.sh \
            
            --broker-list my-kafka-0.my-kafka-headless.kafka-system.svc.cluster.local:9092 \
            --topic test

    CONSUMER:
        kafka-console-consumer.sh \
            
            --bootstrap-server my-kafka.kafka-system.svc.cluster.local:9092 \
            --topic test \
            --from-beginning