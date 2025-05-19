package db
import(
	"context"
	"fmt"
	"log"
	"github.com/jackc/pgx/v5"
)

var Conn *pgx.Conn

func Init(){
	var err error
	connStr := "postgres://postgres:1234@localhost:5432/skillsync"
	Conn, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	fmt.Println("Connected to PostgreSQL...")

}