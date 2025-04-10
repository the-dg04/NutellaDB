package main

import (
	"bytes"
	"db/cli"
	"db/database"
	"github.com/gofiber/fiber/v2"
	"log"
	"path/filepath"
)

func runCLI(args []string) (string, error) {
	var out, errBuf bytes.Buffer
	cli.RootCmd.SetOut(&out)
	cli.RootCmd.SetErr(&errBuf)
	cli.RootCmd.SetArgs(args)
	err := cli.RootCmd.Execute()
	return out.String() + errBuf.String(), err
}

var openDBs = map[string]*database.Database{}

func basePath(dbID string) string {
	return filepath.Join(".", "files", dbID)
}

func getDB(dbID string, createIfMissing bool) (*database.Database, error) {
	if db, ok := openDBs[dbID]; ok {
		return db, nil
	}

	db, err := database.LoadDatabase(basePath(dbID))
	if err != nil {
		if !createIfMissing {
			return nil, err
		}
		db, err = database.NewDatabase(basePath(dbID), dbID)
		if err != nil {
			return nil, err
		}
	}
	openDBs[dbID] = db
	return db, nil
}

func main() {
	app := fiber.New()

	app.Post("/api/create-db", func(c *fiber.Ctx) error {
		var body struct {
			DBID string `json:"dbID"`
		}
		if err := c.BodyParser(&body); err != nil || body.DBID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body must contain dbID"})
		}

		if _, err := getDB(body.DBID, true); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "created", "dbID": body.DBID})
	})

	app.Post("/api/create-collection", func(c *fiber.Ctx) error {
		var body struct {
			DBID  string `json:"dbID"`
			Name  string `json:"name"`
			Order int    `json:"order"`
		}
		if err := c.BodyParser(&body); err != nil || body.DBID == "" || body.Name == "" || body.Order < 3 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "dbID, name and order>=3 required"})
		}

		db, err := getDB(body.DBID, false)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		if err := db.CreateCollection(body.Name, body.Order); err != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "collection created"})
	})

	app.Post("/api/insert", func(c *fiber.Ctx) error {
		var body struct {
			DBID       string `json:"dbID"`
			Collection string `json:"collection"`
			Key        string `json:"key"`
			Value      string `json:"value"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}

		db, err := getDB(body.DBID, false)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll, err := db.GetCollection(body.Collection)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll.InsertKV(body.Key, body.Value)
		return c.JSON(fiber.Map{"status": "inserted"})
	})

	app.Get("/api/find", func(c *fiber.Ctx) error {
		dbID, colName, key := c.Query("dbID"), c.Query("collection"), c.Query("key")
		if dbID == "" || colName == "" || key == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing query params"})
		}

		db, err := getDB(dbID, false)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll, err := db.GetCollection(colName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		val, found := coll.FindKey(key)
		if !found {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "key not found"})
		}
		return c.JSON(fiber.Map{"value": val})
	})

	app.Post("/api/update", func(c *fiber.Ctx) error {
		var body struct {
			DBID       string `json:"dbID"`
			Collection string `json:"collection"`
			Key        string `json:"key"`
			Value      string `json:"value"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
		}
		db, err := getDB(body.DBID, false)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll, err := db.GetCollection(body.Collection)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll.UpdateKV(body.Key, body.Value)
		return c.JSON(fiber.Map{"status": "updated"})
	})

	app.Delete("/api/delete", func(c *fiber.Ctx) error {
		dbID, colName, key := c.Query("dbID"), c.Query("collection"), c.Query("key")
		if dbID == "" || colName == "" || key == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing query params"})
		}

		db, err := getDB(dbID, false)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll, err := db.GetCollection(colName)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		coll.DeleteKey(key)
		return c.JSON(fiber.Map{"status": "deleted (if key existed)"})
	})

	// Git-style routes : NOT Working
	// TODO : Make then work
	app.Post("/api/init", func(c *fiber.Ctx) error {
		var b struct {
			DBID string `json:"dbID"`
		}
		if err := c.BodyParser(&b); err != nil || b.DBID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "dbID required"})
		}
		out, err := runCLI([]string{"init", b.DBID})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": out})
		}
		return c.JSON(fiber.Map{"output": out})
	})

	app.Post("/api/commit-all", func(c *fiber.Ctx) error {
		var b struct{ DBID, Message string }
		if err := c.BodyParser(&b); err != nil || b.DBID == "" || b.Message == "" {
			return c.Status(400).JSON(fiber.Map{"error": "dbID and message required"})
		}
		out, err := runCLI([]string{"commit-all", b.DBID, "-m", b.Message})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": out})
		}
		return c.JSON(fiber.Map{"output": out})
	})

	app.Post("/api/restore", func(c *fiber.Ctx) error {
		var b struct{ DBID string }
		if err := c.BodyParser(&b); err != nil || b.DBID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "dbID required"})
		}
		out, err := runCLI([]string{"restore", b.DBID})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": out})
		}
		return c.JSON(fiber.Map{"output": out})
	})

	app.Post("/api/pack", func(c *fiber.Ctx) error {
		var b struct{ DBID string }
		if err := c.BodyParser(&b); err != nil || b.DBID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "dbID required"})
		}
		out, err := runCLI([]string{"pack", b.DBID})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error(), "output": out})
		}
		return c.JSON(fiber.Map{"output": out})
	})

	log.Println("Fiber listening on :3000")
	if err := app.Listen(":3000"); err != nil {
		log.Fatal(err)
	}
}
