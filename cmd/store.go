package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"

	"github.com/go-sql-driver/mysql"
)

// MySQLStore manages House and Tree data in a MySQL DB.
type MySQLStore struct {
	db    *sql.DB
	stmts map[string]*sql.Stmt
}

const (
	queryGetAllHouses            = "get-all-houses"
	queryGetHouseExistsByHouseID = "get-house-exists-by-house-id"
	queryGetTreesByHouseID       = "get-trees-by-house-id"
	queryAddTreeByHouseID        = "add-tree-by-house-id"
	queryAddHouse                = "add-house"
	queryGetTreeIDsByHouseID     = "get-tree-ids-by-house-id"
	queryFellTreeByTreeID        = "fell-tree-by-tree-id"
	queryGetTreeFallenByTreeID   = "get-tree-fallen-by-tree-id"
	queryRemoveTreeByTreeID      = "delete-tree-by-tree-id"
)

var (
	// ErrDuplicateTree reports an attempt to plant a tree at the exact coordinates and house where one is already growing.
	ErrDuplicateTree = errors.New("tree already growing at given house and absolute location")
	// ErrNoMatchingRecord reports a bad lookup ID.
	ErrNoMatchingRecord = errors.New("no record matching ID")
	// ErrNoTrees reports a tree-less house.
	ErrNoTrees = errors.New("no trees at house")
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
	queryGetHouseExistsByHouseID: `
		SELECT EXISTS (
			SELECT * 
			FROM neighborhood.house h
			WHERE h.id = ?
		);
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
	queryAddHouse: `
		INSERT INTO neighborhood.house (address_one, address_two, city, state, zip)
		VALUES (?, ?, ?, ?, ?);
	`,
	queryAddTreeByHouseID: `
		INSERT INTO neighborhood.tree (house_id, species, x_coord, y_coord, relative_location)
		VALUES (?, ?, ?, ?, ?);
	`,
	queryFellTreeByTreeID: `
		UPDATE neighborhood.tree t
		SET t.fallen = 1
		WHERE t.id = ?;
	`,
	queryGetTreeIDsByHouseID: `
		SELECT
			t.id
		FROM neighborhood.house h
		JOIN neighborhood.tree t ON t.house_id = h.id
		WHERE h.id = ?;
	`,
	queryGetTreeFallenByTreeID: `
		SELECT 
			t.fallen
		FROM neighborhood.tree t
		WHERE t.id = ?;
	`,
	queryRemoveTreeByTreeID: `
		DELETE FROM neighborhood.tree t
		WHERE
			t.id = ?;
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

// GetHouseExistsByHouseID reports whether a given House ID matches a record in the DB.
func (store *MySQLStore) GetHouseExistsByHouseID(houseID int32) (bool, error) {
	stmt := store.stmts[queryGetHouseExistsByHouseID]
	row := stmt.QueryRow(houseID)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("error reading row as whether House exists: %w", err)
	}

	return exists, nil
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

// AddHouse builds a new house in the neighborhood.
func (store *MySQLStore) AddHouse(h *House) error {
	stmt := store.stmts[queryAddHouse]
	result, err := stmt.Exec(h.AddressOne, nullIfEmpty(h.AddressTwo), h.City, h.State, h.Zip)
	if err != nil {
		return fmt.Errorf("INSERT House failed: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error reading last INSERT id: %w", err)
	}

	h.ID = int32(id)
	return nil
}

// AddTreeByHouseID plants a new tree on-site at a specific house.
func (store *MySQLStore) AddTreeByHouseID(t *Tree, houseID int32) error {
	stmt := store.stmts[queryAddTreeByHouseID]
	result, err := stmt.Exec(houseID, t.Species, t.XCoord, t.YCoord, nullIfEmpty(t.RelativeLocation))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return ErrDuplicateTree
		}
		return fmt.Errorf("INSERT Tree failed: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error reading last INSERT id: %w", err)
	}

	t.ID = int32(id)
	return nil
}

// FellRandomTreeByHouseID fells a random tree at a given house.
func (store *MySQLStore) FellRandomTreeByHouseID(houseID int32) error {
	tx, err := store.db.Begin()
	if err != nil {
		return fmt.Errorf("error initiating DB transaction: %w", err)
	}

	rows, err := tx.Stmt(store.stmts[queryGetTreeIDsByHouseID]).Query(houseID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("SELECT Trees failed: %w", err)
	}

	defer rows.Close()

	var treeIDs []int32
	for rows.Next() {
		var treeID int32
		err := rows.Scan(&treeID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to parse row as Tree ID: %w", err)
		}
		treeIDs = append(treeIDs, treeID)
	}

	if len(treeIDs) == 0 {
		tx.Rollback()
		return ErrNoTrees
	}

	randIndex := rand.Intn(len(treeIDs) - 1)

	result, err := tx.Stmt(store.stmts[queryFellTreeByTreeID]).Exec(treeIDs[randIndex])
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to UPDATE Trees: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error reading rows affected: %w", err)
	}

	log.Printf("%v tree(s) felled by storm", rowsAffected)

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	log.Printf("storm damage at house %v committed to DB", houseID)

	return nil
}

// GetTreeFallenByTreeID reports whether a given tree is fallen.
func (store *MySQLStore) GetTreeFallenByTreeID(treeID int32) (bool, error) {
	stmt := store.stmts[queryGetTreeFallenByTreeID]
	row := stmt.QueryRow(treeID)

	var fallen bool
	if err := row.Scan(&fallen); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNoMatchingRecord
		}
		return false, fmt.Errorf("error parsing row as Fallen: %w", err)
	}

	return fallen, nil
}

// RemoveTreeByTreeID removes a tree from a house site.
func (store *MySQLStore) RemoveTreeByTreeID(treeID int32) error {
	stmt := store.stmts[queryRemoveTreeByTreeID]
	result, err := stmt.Exec(treeID)
	if err != nil {
		return fmt.Errorf("error running DELETE Tree: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error reading rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected by DELETE Tree: %w", err)
	}

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

func nullIfEmpty(s string) sql.NullString {
	valid := s != ""
	return sql.NullString{
		String: s,
		Valid:  valid,
	}
}
