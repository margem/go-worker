package main

import (
	"log"
	"os"
	"database/sql"
	"encoding/json"

	"github.com/streadway/amqp"
	_ "gopkg.in/rana/ora.v4"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	db, err := sql.Open("ora", os.Getenv("DB_URL"))
	failOnError(err, "Failed to initiate database")
	defer db.Close()

	err = db.Ping()
	failOnError(err, "Failed to connect to database")

	stmt, err := db.Prepare("update calculo_margem set status = 2 where id = :id")
	failOnError(err, "Failed to prepare SQL statement")

	conn, err := amqp.Dial(os.Getenv("AMQP_URL"))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		os.Getenv("AMQP_QUEUE_NAME"), // name
		true, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil, // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1, // prefetch count
		0, // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"", // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil, // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var dat map[string]interface{}
			err := json.Unmarshal(d.Body, &dat)
			failOnError(err, "Failed to unmarshal JSON")

			res, err := stmt.Exec(dat["ID"])
			failOnError(err, "Failed to execute SQL")

			rowCnt, err := res.RowsAffected()
			failOnError(err, "Failed to get rows affected")

			log.Printf("ID = %.0f, affected = %d", dat["ID"].(float64), rowCnt)

			d.Ack(false)
		}
	}()

	log.Print("Waiting for messages. To exit press CTRL+C")
	<-forever
}