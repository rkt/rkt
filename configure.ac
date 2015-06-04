AC_PREREQ([2.69])
AC_INIT([rkt], [0.5.5+git], [https://github.com/coreos/rkt/issues])

# STAGE1 build settings

## STAGE1:type
AC_ARG_WITH(stage1,
        AS_HELP_STRING([--with-stage1=type],
                [type of stage1 build one of 'src', 'coreos', 'host', 'none' (default: 'coreos')]),
        [RKT_STAGE1_USR_FROM="$withval"],
        [RKT_STAGE1_USR_FROM="coreos"])

case "${RKT_STAGE1_USR_FROM}" in
	none)
        AC_MSG_RESULT([stage1 build disabled])
		;;
	coreos)
        AC_MSG_RESULT([stage1 will reuse coreos image])

        
        # check that some coreos-build specific binaries are here
        AC_CHECK_PROG([WGET],[wget],[wget])
        if [[ -z "$WGET" ]] ; then
            AC_MSG_ERROR([wget not found])
        fi

        AC_CHECK_PROG([GPG],[gpg],[gpg])
        if [[ -z "$GPG" ]] ; then
            AC_MSG_ERROR([gpg not found])
        fi

        AC_CHECK_PROG([MKTEMP],[mktemp],[mktemp])
        if [[ -z "$MKTEMP" ]] ; then
            AC_MSG_ERROR([mktemp not found])
        fi

        AC_CHECK_PROG([MD5SUM],[md5sum],[md5sum])
        if [[ -z "$MD5SUM" ]] ; then
            AC_MSG_ERROR([md5sum not found])
        fi

        AC_CHECK_PROG([CPIO],[cpio],[cpio])
        if [[ -z "$CPIO" ]] ; then
            AC_MSG_ERROR([cpio not found])
        fi

        AC_CHECK_PROG([GZIP],[gzip],[gzip])
        if [[ -z "$GZIP" ]] ; then
            AC_MSG_ERROR([gzip not found])
        fi

        AC_CHECK_PROG([UNSQUASHFS],[unsquashfs],[unsquashfs])
        if [[ -z "$UNSQUASHFS" ]] ; then
            AC_MSG_ERROR([unsquashfs not found])
        fi

        AC_CHECK_PROG([SORT],[sort],[sort])
        if [[ -z "$SORT" ]] ; then
            AC_MSG_ERROR([sort not found])
        fi
        
		;;
	src)
        AC_MSG_RESULT([stage1 will build systemd from source])

        # these are needed for systemd build
                
        AC_CHECK_PROG([INTLTOOLIZE],[intltoolize],[intltoolize])
        if [[ -z "$INTLTOOLIZE" ]] ; then
            AC_MSG_ERROR([intltoolize not installed])
        fi

        AC_CHECK_PROG([LIBTOOLIZE],[libtoolize],[libtoolize])
        if [[ -z "$LIBTOOLIZE" ]] ; then
            AC_MSG_ERROR([libtoolize not installed])
        fi
		;;
	host)
		;;
	*)
		AC_MSG_ERROR([stage1 build type '${RKT_STAGE1_USR_FROM}' unsupported])
esac

AC_SUBST(RKT_STAGE1_USR_FROM)

## STAGE1: Systemd git path
AC_ARG_WITH(stage1-systemd-src,
        AS_HELP_STRING([--with-stage1-systemd-src=git-path],
                [address to git repository of systemd, used in 'src' build mode (default: https://github.com/systemd/systemd.git)]),
        [RKT_STAGE1_SYSTEMD_SRC="$withval"],
        [RKT_STAGE1_SYSTEMD_SRC='https://github.com/systemd/systemd.git'])

AC_SUBST(RKT_STAGE1_SYSTEMD_SRC)

## STAGE1: Systemd version
AC_ARG_WITH(stage1-systemd-version,
        AS_HELP_STRING([--with-stage1-systemd-version=version],
                [systemd version to build (default: v220)]),
        [RKT_STAGE1_SYSTEMD_VER="$withval"],
        [RKT_STAGE1_SYSTEMD_VER="v220"])

AC_SUBST(RKT_STAGE1_SYSTEMD_VER)

## STAGE1: linker-defined custom STAGE1 image path, default is unset
AC_ARG_WITH(stage1-image-path,
        AS_HELP_STRING([--with-stage1-image-path],
                [custom stage1 image path (default: "")]),
        [RKT_STAGE1_IMAGE="$withval"],
        [RKT_STAGE1_IMAGE=""])

RKT_STAGE1_IMAGE_FLAGS=
# if stage1 image variable is set, add a linker flag to rkt defining the variable
if [[ -n "$RKT_STAGE1_IMAGE" ]] ; then
   RKT_STAGE1_IMAGE_FLAGS="-ldflags \"-X main.defaultStage1Image '${RKT_STAGE1_IMAGE}'\""
fi

AC_SUBST(RKT_STAGE1_IMAGE_FLAGS)


# Checks for programs.
AC_PROG_CC
AC_PROG_INSTALL

AC_CHECK_PROG([GIT],[git],[git])
if [[ -z "${GIT}" ]] ; then
    AC_MSG_ERROR([git not installed])
fi

AC_CHECK_PROG([GOBINARY],[go],[go])
if [[ -z "${GOBINARY}" ]] ; then
    AC_MSG_ERROR([Go language binary not found.])
fi

AC_CHECK_PROG([GOFMTBINARY],[gofmt],[gofmt])
if [[ -z "${GOFMTBINARY}" ]] ; then
    AC_MSG_ERROR([gofmt binary not found.])
fi

AC_CHECK_PROG([BASH],[bash],[bash])
if [[ -z "${BASH}" ]] ; then
    AC_MSG_ERROR([bash not found])
fi

AC_CONFIG_FILES([Makefile])
AC_OUTPUT
