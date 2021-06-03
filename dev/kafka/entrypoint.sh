#!/bin/sh

# setup SASL
if [ -n "$KAFKA_USERNAME" -a -n "$KAFKA_PASSWORD" ] ; then
    echo "Configuring SASL 1/2."
    cat <<EOF > /etc/kafka/kafka_server_jaas.conf
Client {
    org.apache.kafka.common.security.scram.ScramLoginModule required
    username="$KAFKA_USERNAME"
    password="$KAFKA_PASSWORD";
};
KafkaServer {
    org.apache.kafka.common.security.scram.ScramLoginModule required
    username="$KAFKA_USERNAME"
    password="$KAFKA_PASSWORD";
};
EOF
fi

/setup.sh &

exec /etc/confluent/docker/run
