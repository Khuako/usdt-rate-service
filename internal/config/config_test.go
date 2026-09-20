package config

import (
	"errors"
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	testConfig := Config{
		DatabaseURL: "postgres://localhost/test", GRPCAddr: ":6000",
		KrakenBaseURL: "example.com",
	}
	tests := []struct {
		name    string
		env     map[string]string
		args    []string
		want    Config
		wantErr error
		anyErr  bool
	}{
		{name: "Defaults", env: map[string]string{"DATABASE_URL": "postgres://localhost/test"}, args: []string{}, want: Config{
			DatabaseURL: "postgres://localhost/test", GRPCAddr: ":50051",
			KrakenBaseURL: "https://api.kraken.com",
		}},
		{name: "all env", env: map[string]string{"DATABASE_URL": "postgres://localhost/test", "GRPC_ADDR": ":6000", "KRAKEN_BASE_URL": "example.com"}, want: testConfig},
		{name: "all flags", args: []string{"-grpc-addr=:6000", "-database-url=postgres://localhost/test", "-kraken-base-url=example.com"}, want: testConfig},
		{
			name: "env and flags",
			env:  map[string]string{"DATABASE_URL": "1", "GRPC_ADDR": "431431", "KRAKEN_BASE_URL": "431431"},
			args: []string{"-grpc-addr=:6000", "-database-url=postgres://localhost/test", "-kraken-base-url=example.com"},
			want: testConfig,
		},
		{
			name: "empty env and flag",
			env:  map[string]string{"DATABASE_URL": ""},
			args: []string{"-grpc-addr=:6000", "-database-url=postgres://localhost/test", "-kraken-base-url=example.com"},
			want: testConfig,
		},
		{
			name:    "no db url",
			wantErr: ErrEmptyDB,
		},
		{name: "invalid db address", args: []string{"-grpc-addr=:6000", "-database-url=", "-kraken-base-url=example.com"}, wantErr: ErrEmptyDB},
		{name: "invalid grpc", args: []string{"-grpc-addr=", "-database-url=postgres://localhost/test", "-kraken-base-url=example.com"}, wantErr: ErrEmptyGRPCAddr},
		{name: "invalid base url", args: []string{"-grpc-addr=:6000", "-database-url=postgres://localhost/test", "-kraken-base-url="}, wantErr: ErrEmptyKrakenBaseUrl},
		{
			name:   "empty flag",
			env:    map[string]string{"DATABASE_URL": "postgres://localhost/test"},
			args:   []string{"-grpc-addr=:6000", "-database-url=", "-kraken-base-url=example.com"},
			anyErr: true,
		},

		{
			name:   "flag without an argument",
			env:    map[string]string{"DATABASE_URL": "postgres://localhost/test"},
			args:   []string{"-grpc-addr"},
			anyErr: true,
		},
		{
			name:    "spaces in flags",
			args:    []string{"-grpc-addr=    ", "-database-url=postgres://local host/test", "-kraken-base-url=exam ple.com"},
			wantErr: ErrEmptyGRPCAddr,
		},
		{
			name:   "unknown flag",
			env:    map[string]string{"DATABASE_URL": "postgres://localhost/test"},
			args:   []string{"-grpc-addfar=4321"},
			anyErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, name := range []string{"DATABASE_URL", "GRPC_ADDR", "KRAKEN_BASE_URL"} {
				t.Setenv(name, "")
				if err := os.Unsetenv(name); err != nil {
					t.Fatal(err)
				}
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got, err := Load(tt.args)
			if tt.anyErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got wrong error, got: %v, want: %v", got, tt.wantErr)
			}

			if got.KrakenBaseURL != tt.want.KrakenBaseURL {
				t.Errorf("got wrong baseUrl, got: %v, want: %v", got.KrakenBaseURL, tt.want.KrakenBaseURL)
			}
			if got.DatabaseURL != tt.want.DatabaseURL {
				t.Errorf("got wrong DatabaseURL, got: %v, want: %v", got.DatabaseURL, tt.want.DatabaseURL)
			}
			if got.GRPCAddr != tt.want.GRPCAddr {
				t.Errorf("got wrong GRPCAddr, got: %v, want: %v", got.GRPCAddr, tt.want.GRPCAddr)
			}
		})

	}
}
