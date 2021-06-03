#!/bin/sh

# create topics with multiple partitions for scaling
for topic in "platform.inventory.events" "patchman.evaluator.upload" \
             "patchman.evaluator.recalc" "platform.remediation-updates.patch" "test"
do
    until /usr/bin/kafka-topics --create --if-not-exists --topic $topic --partitions 1 --zookeeper zookeeper:2181 \
    --replication-factor 1; do
      echo "Unable to create topic $topic"
      sleep 1
    done
    echo "Topic $topic created successfully"
done

# setup SASL
if [ -n "$KAFKA_USERNAME" -a -n "$KAFKA_PASSWORD" ] ; then
    echo "Configuring SASL 2/2."
    kafka-configs --zookeeper localhost:2181 --alter --add-config "SCRAM-SHA-512=[password=$KAFKA_PASSWORD]" --entity-type users --entity-name "$KAFKA_USERNAME"
else
    echo "KAFKA_USERNAME or KAFKA_PASSWORD not set. Skipping SASL setup."
fi
