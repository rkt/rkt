# Inputs cleared:
# GR_TARGET
# GR_PREREQS
# GR_SRCDIR
# GR_BRANCH

# Inputs left alone:
# GIT
# REVSDIR

_GR_ESCAPED_PREREQS_ := $(foreach p,$(GR_PREREQS),$(call escape-for-file,$p))
_GR_SPACE_ :=
_GR_SPACE_ +=
_GR_PREREQS_ := $(subst $(_GR_SPACE_),/,$(_GR_ESCAPED_PREREQS_))
_GR_ESCAPED_TARGET_ := $(call escape-for-file,$(GR_TARGET))

_GR_REVDIR_ := $(REVSDIR)/$(_GR_ESCAPED_TARGET_)/$(_GR_PREREQS_)
# To avoid too long filenames, rev file is stored inside
# <builddir>/revs/<escaped_target>/<escaped_prereq_1>/<escaped_prereq_2>/.../<escaped_prereq_last>/git-rev
_GR_REVFILE_ := $(_GR_REVDIR_)/git-rev
_GR_REVFILE_TMP_ := $(_GR_REVFILE_).tmp
_GR_REVFILE_REFRESH_TARGET_ := $(_GR_REVDIR_)/REVFILE_REFRESH

CREATE_DIRS += $(_GR_REVDIR_)

$(GR_TARGET): $(_GR_REVFILE_)

$(_GR_REVFILE_): $(_GR_REVFILE_REFRESH_TARGET_)

$(call forward-vars,$(_GR_REVFILE_REFRESH_TARGET_), \
	GIT GR_SRCDIR GR_BRANCH _GR_REVFILE_TMP_ _GR_REVFILE_)
$(_GR_REVFILE_REFRESH_TARGET_): $(GR_PREREQS) | $(_GR_REVDIR_)
	set -e; \
	"$(GIT)" -C "$(GR_SRCDIR)" fetch origin "$(GR_BRANCH)"; \
	"$(GIT)" -C "$(GR_SRCDIR)" rev-parse "origin/$(GR_BRANCH)" >"$(_GR_REVFILE_TMP_)"; \
	if cmp --silent "$(_GR_REVFILE_)" "$(_GR_REVFILE_TMP_)"; then \
		rm -f "$(_GR_REVFILE_TMP_)"; \
	else \
		"$(GIT)" -C "$(GR_SRCDIR)" reset --hard "origin/$(GR_BRANCH)"; \
		mv "$(_GR_REVFILE_TMP_)" "$(_GR_REVFILE_)"; \
	fi

.PHONY: $(_GR_REVFILE_REFRESH_TARGET_)

$(call undefine-namespaces,GR _GR)
