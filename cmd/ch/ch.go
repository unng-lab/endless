package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"

	_ "github.com/ClickHouse/clickhouse-go"
)

func main() {
	// Подключение к ClickHouse
	db, err := sql.Open("clickhouse", "tcp://192.168.1.156:9000?username=default&password=PrimaBek123&database=default")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Запрос данных
	rows, err := db.Query("SELECT * FROM units LIMIT 10000")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Создание CSV файла
	file, err := os.Create("output01.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Запись заголовков в CSV
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}
	if err := writer.Write(columns); err != nil {
		log.Fatal(err)
	}

	// Запись данных в CSV
	for rows.Next() {
		columnsLength := len(columns)
		values := make([]interface{}, columnsLength)
		valuePointers := make([]interface{}, columnsLength)
		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			log.Fatal(err)
		}

		record := make([]string, columnsLength)
		for i, val := range values {
			if b, ok := val.([]byte); ok {
				record[i] = string(b)
			} else {
				record[i] = fmt.Sprintf("%v", val)
			}
		}

		if err := writer.Write(record); err != nil {
			log.Fatal(err)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Данные успешно записаны в output.csv")
}
