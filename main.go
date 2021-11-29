package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

/*
	SELECT id, created_at, username, custom_id, tags, ws_id, "type"
	FROM public.consumers;
*/
type Consumer struct {
	Id        uuid.UUID `db:"id" json:"id" validate:"required,Id" gorm:"type:uuid;primaryKey;"`
	CreatedAt time.Time `db:"created_at" json:"created_at" validate:"required,CreatedAt"`
	Username  string    `db:"username" json:"username" validate:"required,username"`
	CustomId  string    `db:"custom_id" json:"custom_id"`
	WSId      uuid.UUID `db:"ws_id" json:"ws_id" validate:"required,ws_id" gorm:"type:uuid"`
}

/*
	SELECT id, created_at, "name", consumer_id, service_id, route_id, config, enabled, cache_key, protocols, tags, ws_id
	FROM public.plugins;
*/
type Plugin struct {
	Id         uuid.UUID `db:"id" json:"id" validate:"required,Id" gorm:"type:uuid;primaryKey;"`
	CreatedAt  time.Time `db:"created_at" json:"created_at" validate:"required,CreatedAt"`
	Name       string    `db:"name" json:"name" validate:"required,name"`
	ConsumerId uuid.UUID `db:"consumer_id" json:"consumer_id" validate:"required,consumer_id" gorm:"type:uuid"`
	ServiceId  uuid.UUID `db:"service_id" json:"service_id" validate:"required,service_id" gorm:"type:uuid"`
	Config     string    `db:"config" json:"config" validate:"required,config"`
	Enabled    bool      `db:"enabled" json:"enabled" validate:"required,enabled"`
	CacheKey   string    `db:"cache_key" json:"cache_key" validate:"required,cache_key"`
	Protocols  string    `db:"protocols" json:"protocols" validate:"required,protocols"`
	WSId       uuid.UUID `db:"ws_id" json:"ws_id" validate:"required,ws_id" gorm:"type:uuid"`
}

/*
	SELECT id, created_at, updated_at, "name", retries, protocol, host, port, "path",
	connect_timeout, write_timeout, read_timeout, tags, client_certificate_id,
	tls_verify, tls_verify_depth, ca_certificates, ws_id
	FROM public.services;
*/
type Service struct {
	Id             uuid.UUID `db:"id" json:"id" validate:"required,Id" gorm:"type:uuid;primaryKey;"`
	CreatedAt      time.Time `db:"created_at" json:"created_at" validate:"required,CreatedAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at" validate:"required,UpdatedAt"`
	Name           string    `db:"name" json:"name"`
	Retries        int       `db:"retries" json:"retries" validate:"required,retries"`
	Protocol       string    `db:"protocol" json:"protocol" validate:"required,protocol"`
	Host           string    `db:"host" json:"host" validate:"required,host"`
	Port           int       `db:"port" json:"port" validate:"required,port"`
	Path           string    `db:"path" json:"path" validate:"required,path"`
	ConnectTimeout int64     `db:"connect_timeout" json:"connect_timeout" validate:"required,connect_timeout"`
	WriteTimeout   int64     `db:"write_timeout" json:"write_timeout" validate:"required,write_timeout"`
	ReadTimeout    int64     `db:"read_timeout" json:"read_timeout" validate:"required,read_timeout"`
	WSId           uuid.UUID `db:"ws_id" json:"ws_id" validate:"required,ws_id" gorm:"type:uuid"`
}

/*
	id                  |  name   | comment |       created_at       | meta | config
*/
type Workspace struct {
	Id   uuid.UUID `db:"id" json:"id" validate:"required,Id" gorm:"type:uuid;primaryKey;"`
	Name string    `db:"name" json:"name"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Gather info
	redisHost := flag.String("redishost", "127.0.0.1", "Redis hostname for Rate Limiting")
	redisPort := flag.String("redisport", "6379", "Redit port for Rate Limiting")
	redisDictionary := flag.String("redisdictionary", "kong_rate_limiting_counters", "Redis dictionary name")
	redisNamespace := flag.String("redisnamespace", "L12Tt6QKCod1KLmT30RAz6GUj0KzVCp1", "Redis namespace")
	redisSyncRate := flag.Int("redissyncrate", 50, "Redis Sync Rate")
	redisUseSsl := flag.Bool("redisusessl", false, "Use SSL for redis connection")
	workspace := flag.String("workspace", "default", "Override the default workspace to load into")
	createServices := flag.Bool("createservices", false, "Create services for use in RLA attachment, false will use 20 random existing services")
	serviceCount := flag.Int("servicecount", 20, "Uses this many services, at random, for attachment")
	maxRateLimits := flag.Int("plugincount", 520000, "Number of Rate Limiting Advanced plugins to create in total") // stops at-or-just-above this count

	flag.Parse()

	pluginConfig := fmt.Sprintf(`{"path": null, "limit": [5], "redis": {"ssl": %v, "host": null, "port": null, "timeout": 2000, "database": 0, "password": null, "ssl_verify": false, "server_name": null, "read_timeout": null, "send_timeout": null, "sentinel_role": null, "connect_timeout": null, "sentinel_master": null, "cluster_addresses": ["%s:%s"], "keepalive_backlog": null, "sentinel_password": null, "sentinel_addresses": null, "keepalive_pool_size": 30}, "strategy": "redis", "namespace": "%s", "sync_rate": %d, "identifier": "service", "header_name": null, "window_size": [30], "window_type": "sliding", "dictionary_name": "%s", "hide_client_headers": false, "retry_after_jitter_max": 0}`, *redisUseSsl, *redisHost, *redisPort, *redisNamespace, *redisSyncRate, *redisDictionary)
	pluginProtocols := `{grpc,grpcs,http,https}`
	pluginName := "rate-limiting-advanced"

	if *serviceCount < 2 {
		fmt.Println("servicecount must be greater than 1")
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(fmt.Sprintf("postgresql://%s:%s@%s/%s?connect_timeout=10", os.Getenv("KONG_PG_USER"), os.Getenv("KONG_PG_PASSWORD"), os.Getenv("KONG_PG_HOST"), os.Getenv("KONG_PG_DATABASE"))), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Get the chosen workspace ID
	var workspaceObject Workspace
	result := db.Where(Workspace{Name: *workspace}).First(&workspaceObject)
	if result.RowsAffected < 1 {
		fmt.Printf("Workspace %s was not found", *workspace)
		os.Exit(1)
	}

	fmt.Printf("Workspace name %s was found, ID is: %s\n", *workspace, workspaceObject.Id)

	// Set up globals
	totalRateLimits := 0
	pageLimits := 0

	var services []Service
	if *createServices {
		fmt.Printf("-createservices is set, creating %d new services\n", *serviceCount)

		tx := db.Begin()
		createdAt := time.Now()

		for i := 0; i < *serviceCount; i++ {
			services = append(services, Service{
				Id:             uuid.New(),
				CreatedAt:      createdAt,
				UpdatedAt:      createdAt,
				Name:           RandStringRunes(10),
				Retries:        5,
				Protocol:       "https",
				Host:           "mockbin.org",
				Port:           443,
				Path:           "/request",
				ConnectTimeout: 60000,
				WriteTimeout:   60000,
				ReadTimeout:    60000,
				WSId:           workspaceObject.Id,
			})
		}
		tx.Create(services)
		tx.Commit()
	} else {
		// Get 20 service IDs at random
		result := db.Limit(*serviceCount).Find(&services)

		if result.RowsAffected != int64(*serviceCount) {
			fmt.Printf("Services query only returned %d rows, when %d were requested, use -createservices argument to automatically create more\n", result.RowsAffected, *serviceCount)
			os.Exit(1)
		}
	}

	var tx *gorm.DB = db.Begin()
	for totalRateLimits <= *maxRateLimits {
		if pageLimits > 9999 { // commit the transaction page every 10000 rows
			pageLimits = 0
			tx.Commit()
			tx = db.Begin()
		}

		fmt.Printf(">> REMAINING PLUGINS TO INSERT: %d <<\n\n", *maxRateLimits-totalRateLimits)
		createdAt := time.Now()

		// First create a new Consumer
		consumer := Consumer{
			Id:        uuid.New(),
			CreatedAt: createdAt,
			Username:  RandStringRunes(10),
			CustomId:  RandStringRunes(12),
			WSId:      workspaceObject.Id,
		}

		result := tx.Create(consumer)

		if result.Error == nil && result.RowsAffected > 0 {
			// Attach a random (1 to 20) number of rate limiting plugins to this consumer
			numberToInsert := rand.Intn(*serviceCount) + 1
			pluginsToInsert := []Plugin{}

			for i := 0; i < numberToInsert; i++ {
				newPlugin := Plugin{
					Id:         uuid.New(),
					CreatedAt:  createdAt,
					Name:       pluginName,
					ConsumerId: consumer.Id,
					ServiceId:  services[i].Id,
					Config:     pluginConfig,
					Enabled:    true,
					CacheKey:   fmt.Sprintf("plugins:%s::%s:%s::%s", pluginName, services[i].Id, consumer.Id, workspaceObject.Id),
					Protocols:  pluginProtocols,
					WSId:       workspaceObject.Id,
				}

				pluginsToInsert = append(pluginsToInsert, newPlugin)
			}

			result := tx.Create(&pluginsToInsert)
			if result.Error != nil {
				fmt.Printf("Error inserting plugins: %s", result.Error.Error())
				os.Exit(1)
			}
			totalRateLimits += int(result.RowsAffected)
			pageLimits += int(result.RowsAffected)
		}
	}
	tx.Commit()
}
