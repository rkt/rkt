/* just enough ELF support for finding the interpreter in the program header
 * table, this should theoretically work as-is on both big-endian and
 * little-endian
 */

/* values of interest */
#define ELF_BITS_32	0x1
#define ELF_BITS_64	0x2
#define ELF_ENDIAN_LITL	0x1
#define ELF_ENDIAN_BIG	0x2
#define ELF_PT_INTERP	0x3

/* offsets of interest */
#define ELF_BITS	0x4
#define ELF_ENDIAN	0x5
#define ELF_VERSION	0x6
#define ELF32_PHT_OFF	0x1c
#define ELF32_PHTE_SIZE	0x2a
#define ELF32_PHTE_CNT	0x2c
#define ELF32_PHE_OFF	0x4
#define ELF32_PHE_SIZE	0x10
#define ELF64_PHT_OFF	0x20
#define ELF64_PHTE_SIZE	0x36
#define ELF64_PHTE_CNT	0x38
#define ELF64_PHE_OFF	0x8
#define ELF64_PHE_SIZE	0x20

/* multibyte value accessors, choose which based on ELF_BITS and ELF_ENDIAN */

#define SHIFT(_val, _bytes) ((unsigned long long)(_val) << ((_bytes) * 8))
static uint64_t le32_lget(const uint8_t *addr)
{
	uint64_t val = 0;
	val += SHIFT(addr[3], 3);
	val += SHIFT(addr[2], 2);
	val += SHIFT(addr[1], 1);
	val += SHIFT(addr[0], 0);
	return val;
}
static uint64_t be32_lget(const uint8_t *addr)
{
	uint64_t val = 0;
	val += SHIFT(addr[0], 3);
	val += SHIFT(addr[1], 2);
	val += SHIFT(addr[2], 1);
	val += SHIFT(addr[3], 0);
	return val;
}

static uint64_t le64_lget(const uint8_t *addr)
{
	uint64_t val = 0;
	val += SHIFT(addr[7], 7);
	val += SHIFT(addr[6], 6);
	val += SHIFT(addr[5], 5);
	val += SHIFT(addr[4], 4);
	val += SHIFT(addr[3], 3);
	val += SHIFT(addr[2], 2);
	val += SHIFT(addr[1], 1);
	val += SHIFT(addr[0], 0);
	return val;
}
static uint64_t be64_lget(const uint8_t *addr)
{
	uint64_t val = 0;
	val += SHIFT(addr[0], 7);
	val += SHIFT(addr[1], 6);
	val += SHIFT(addr[2], 5);
	val += SHIFT(addr[3], 4);
	val += SHIFT(addr[4], 3);
	val += SHIFT(addr[5], 2);
	val += SHIFT(addr[6], 1);
	val += SHIFT(addr[7], 0);
	return val;
}
static uint32_t le_iget(const uint8_t *addr)
{
	return (uint32_t)le32_lget(addr);
}
static uint32_t be_iget(const uint8_t *addr)
{
	return (uint32_t)be32_lget(addr);
}
static uint16_t le_sget(const uint8_t *addr)
{
	uint16_t val = 0;
	val += SHIFT(addr[1], 1);
	val += SHIFT(addr[0], 0);
	return val;
}
static uint16_t be_sget(const uint8_t *addr)
{
	uint16_t val = 0;
	val += SHIFT(addr[0], 0);
	val += SHIFT(addr[1], 1);
	return val;
}
