package setup

import "testing"

func TestOptionsValidate(t *testing.T) {
	valid := Options{
		Port:       2053,
		Path:       "/private/admin/panel",
		Username:   "admin_user",
		Password:   "secret12",
		AccessMode: "ip",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	domain := valid
	domain.AccessMode = "domain"
	domain.Domain = "panel.example.com"
	domain.Email = "admin@example.com"
	if err := domain.Validate(); err != nil {
		t.Fatalf("Validate domain: %v", err)
	}

	for name, mutate := range map[string]func(*Options){
		"port":     func(o *Options) { o.Port = 70000 },
		"path":     func(o *Options) { o.Path = "/bad//path" },
		"username": func(o *Options) { o.Username = "root" },
		"password": func(o *Options) { o.Password = "123" },
		"mode":     func(o *Options) { o.AccessMode = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			options := valid
			mutate(&options)
			if err := options.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
