AC_INIT([pg_maintenance], [0.1], [bixority@gmail.com])
AC_PREREQ([2.72]) # Minimum required autoconf version
AM_INIT_AUTOMAKE([foreign])

# Check for a C compiler
AC_PROG_CC

CFLAGS="$CFLAGS -Os"

AC_ARG_ENABLE([static],
    [AS_HELP_STRING([--enable-static], [Build static executable])],
    [],
    [enable_static=yes]
)

# Use pkg-config to find libpq
PKG_CHECK_MODULES_STATIC([LIBPQ], [libpq], [], [AC_MSG_ERROR([libpq not found!])])

LDFLAGS="$LDFLAGS -static -static-libgcc"


# Define a standard Makefile
AC_CONFIG_FILES([
    Makefile
    src/Makefile
])
AC_OUTPUT
