package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"
)

// MySQLStore manages House and Tree data in a MySQL DB.
type MySQLStore struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
}

const (
	queryGetAllHouses      = "get-all-houses"
	queryGetTreesByHouseID = "get-trees-by-house-id"
	queryAddTreeByHouseID  = "add-tree-by-house-id"
)

var (
	// ErrDuplicateTree reports an attempt to plant a tree at the exact coordinates and house where one is already growing.
	ErrDuplicateTree = errors.New("tree already growing at given house and absolute location")
)

var unprepared = map[string]string{
	queryGetAllHouses: `
		SELECT 
			h.id,
			h.address_one,
			h.address_two,
			h.city,
			h.state,
			h.zip
		FROM neighborhood.house h
		ORDER BY h.id ASC;
	`,
	queryGetTreesByHouseID: `
		SELECT 
			t.id, 
			t.species, 
			t.x_coord, 
			t.y_coord, 
			t.relative_location, 
			t.fallen
		FROM neighborhood.tree t
		WHERE 
			t.house_id = ?
		ORDER BY t.id ASC;
	`,
	queryAddTreeByHouseID: `
		INSERT INTO neighborhood.tree (house_id, species, x_coord, y_coord, relative_location)
		VALUES (?, ?, ?, ?, ?);
	`,
}

// NewMySQLStore returns a store with backing DB connection and statements prepared against it.
func NewMySQLStore(db *sql.DB) (*MySQLStore, error) {
	stmts := make(map[string]*sql.Stmt)
	for key, query := range unprepared {
		stmt, err := db.Prepare(query)
		if err != nil {
			return nil, fmt.Errorf("error preparing statement %s: %s", key, err)
		}
		stmts[key] = stmt
	}
	store := MySQLStore{
		db:    db,
		stmts: stmts,
	}

	return &store, nil
}

// GetAllHouses lists all Houses in the neighborhood.
func (store *MySQLStore) GetAllHouses() ([]House, error) {
	stmt := store.stmts[queryGetAllHouses]

	rows, err := stmt.Query()
	if err != nil {
		return nil, fmt.Errorf("SELECT Houses failed: %w", err)
	}

	defer rows.Close()

	var houses []House
	for rows.Next() {
		var house House
		var addressTwo sql.NullString
		err := rows.Scan(
			&house.ID,
			&house.AddressOne,
			&addressTwo,
			&house.City,
			&house.State,
			&house.Zip,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to parse row as House: %w", err)
		}
		house.AddressTwo = emptyIfNull(addressTwo)
		houses = append(houses, house)
	}

	return houses, nil
}

// GetTreesByHouseID lists trees for a given House ID.
func (store *MySQLStore) GetTreesByHouseID(houseID int32) ([]Tree, error) {
	stmt := store.stmts[queryGetTreesByHouseID]

	rows, err := stmt.Query(houseID)
	if err != nil {
		return nil, fmt.Errorf("SELECT Trees failed: %w", err)
	}

	defer rows.Close()

	var trees []Tree
	for rows.Next() {
		var tree Tree
		var relativeLocation sql.NullString
		err := rows.Scan(
			&tree.ID,
			&tree.Species,
			&tree.XCoord,
			&tree.YCoord,
			&relativeLocation,
			&tree.Fallen,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to parse row as Tree: %w", err)
		}
		tree.RelativeLocation = emptyIfNull(relativeLocation)
		trees = append(trees, tree)
	}

	return trees, nil
}

// AddTreeByHouseID plants a new tree on-site at a specific house.
func (store *MySQLStore) AddTreeByHouseID(t *Tree, houseID int32) error {
	stmt := store.stmts[queryAddTreeByHouseID]

	result, err := stmt.Exec(houseID, t.Species, t.XCoord, t.YCoord, t.RelativeLocation)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrDuplicateTree
		}
		return fmt.Errorf("INSERT Tree failed: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error reading last INSERT id")
	}

	t.ID = int32(id)
	return nil
}

// Close cleans up prepared statements.
func (store *MySQLStore) Close() {
	log.Println("store: closing prepared statements")
	for key, stmt := range store.stmts {
		var err error
		if err = stmt.Close(); err != nil {
			log.Printf("store: failed to close stmt %s: %s", key, err)
		}
	}
}

func emptyIfNull(nullString sql.NullString) string {
	if nullString.Valid {
		return nullString.String
	}
	return ""
}
