package pg

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"

	"strings"
	"database/sql"
	
	//"fmt"

	"github.com/pkg/errors"
	"github.com/jmoiron/sqlx"
	sq "gopkg.in/Masterminds/squirrel.v1"

	"github.com/micromdm/micromdm/pkg/crypto"
	"github.com/micromdm/micromdm/platform/config"
	"github.com/micromdm/micromdm/platform/pubsub"
)

type Postgres struct{ db *sqlx.DB }

func NewDB(db *sqlx.DB, sub pubsub.Subscriber) (*Postgres, error) {
	// Required for TIMESTAMP DEFAULT 0
	_,err := db.Exec(`SET sql_mode = '';`)
	
	_,err = db.Exec(`CREATE TABLE IF NOT EXISTS server_config (
			config_id INT PRIMARY KEY,
		    push_certificate bytea DEFAULT NULL,
		    private_key bytea DEFAULT NULL
		);`)
	if err != nil {
	   return nil, errors.Wrap(err, "creating server_config sql table failed")
	}
	
	_,err = db.Exec(`CREATE TABLE IF NOT EXISTS dep_tokens (
			consumer_key VARCHAR(36) PRIMARY KEY,
			consumer_secret TEXT NULL,
			access_token TEXT NULL,
			access_secret TEXT NULL,
		    access_token_expiry TIMESTAMPTZ DEFAULT '1970-01-01 00:00:00+00'
		);`)
	if err != nil {
	   return nil, errors.Wrap(err, "creating dep_tokens sql table failed")
	}

	store := &Postgres{db: db}
	return store, err
}

func columns() []string {
	return []string{
		"push_certificate",
		"private_key",
	}
}

const tableName = "server_config"

func (d *Postgres) SavePushCertificate(ctx context.Context, cert []byte, key []byte) error {
	updateQuery, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Update(tableName).
		Prefix("ON CONFLICT (config_id) DO").
		Set("config_id", 1).
		Set("push_certificate", cert).
		Set("private_key", key).
		ToSql()
		
	if err != nil {
		return errors.Wrap(err, "building update query for push_info save")
	}
	updateQuery = strings.Replace(updateQuery, tableName, "", -1)

	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Insert(tableName).
		Columns("config_id", "push_certificate", "private_key").
		Values(
			1,
			cert,
			key,
		).
		Suffix(updateQuery).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building push_info save query")
	}

	_, err = d.db.ExecContext(ctx, query, args...)
	return errors.Wrap(err, "exec push_info save in pg")
}

func (d *Postgres) serverConfig(ctx context.Context) (*config.ServerConfig, error) {
	query, args, err := sq.StatementBuilder.
		PlaceholderFormat(sq.Dollar).
		Select(columns()...).
		From(tableName).
		Where(sq.Eq{"config_id": 1}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "building sql")
	}

	var config config.ServerConfig
	
	err = d.db.QueryRowxContext(ctx, query, args...).StructScan(&config)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, serverConfigNotFoundErr{}
	}
	return &config, errors.Wrap(err, "finding config by config_id")
}



func (d *Postgres) GetPushCertificate(ctx context.Context) ([]byte, error) {
	cert, err := d.PushCertificate(ctx)
	if err != nil {
		return nil, err
	}
	if len(cert.Certificate) > 0 {
		return cert.Certificate[0], nil
	}
	return nil, nil
}

func (d *Postgres) PushCertificate(ctx context.Context) (*tls.Certificate, error) {
	conf, err := d.serverConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get server config for push cert")
	}

	// load private key
	pkeyBlock, _ := pem.Decode(conf.PrivateKey)
	if pkeyBlock == nil {
		return nil, errors.New("decode private key for push cert")
	}

	priv, err := x509.ParsePKCS1PrivateKey(pkeyBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse push certificate key from server config")
	}

	// load certificate
	certBlock, _ := pem.Decode(conf.PushCertificate)
	if certBlock == nil {
		return nil, errors.New("decode push certificate PEM")
	}

	pushCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse push certificate from server config")
	}

	cert := tls.Certificate{
		Certificate: [][]byte{pushCert.Raw},
		PrivateKey:  priv,
		Leaf:        pushCert,
	}
	return &cert, nil
}

func (d *Postgres) PushTopic(ctx context.Context) (string, error) {
	cert, err := d.PushCertificate(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get push certificate for topic")
	}
	topic, err := crypto.TopicFromCert(cert.Leaf)
	return topic, errors.Wrap(err, "get topic from push certificate")
}

type serverConfigNotFoundErr struct{}

func (e serverConfigNotFoundErr) Error() string  { return "server_config not found" }
func (e serverConfigNotFoundErr) NotFound() bool { return true }
