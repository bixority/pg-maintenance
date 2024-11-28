bin_PROGRAMS = pg_maintenance
pg_maintenance_SOURCES = main.c
pg_maintenance_CFLAGS = $(LIBPQ_CFLAGS) $(CFLAGS)
pg_maintenance_LDADD = $(LIBPQ_LIBS) -lpq -lssl -lcrypto -lz -lpthread -ldl -lm
