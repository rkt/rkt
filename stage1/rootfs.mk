_S1_RF_NAME_ := usr_from_$(S1_RF_FLAVOR)

INSTALL_DIRS += \
	$(ACIDIR):- \
	$(ACIROOTFSDIR):0750

$(call inc-one,$(_S1_RF_NAME_)/$(_S1_RF_NAME_).mk)

$(call undefine-namespaces,S1_RF _S1_RF)
