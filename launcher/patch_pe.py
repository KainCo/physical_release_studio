# Setzt IMAGE_FILE_REMOVABLE_RUN_FROM_SWAP (0x0400) und IMAGE_FILE_NET_RUN_FROM_SWAP (0x0800).
# Windows kopiert die EXE dann beim Start vom Datentraeger in die Auslagerungsdatei -> Disc-Wechsel ist unkritisch.
# Aufruf nach dem Build: python patch_pe.py GOGAutoLauncher.exe
import struct, sys
p = sys.argv[1]
d = bytearray(open(p, 'rb').read())
pe = struct.unpack_from('<I', d, 0x3c)[0]
assert d[pe:pe+4] == b'PE\0\0'
ch = struct.unpack_from('<H', d, pe + 22)[0]
struct.pack_into('<H', d, pe + 22, ch | 0x0400 | 0x0800)
open(p, 'wb').write(d)
print('Characteristics: %#06x -> %#06x' % (ch, ch | 0x0C00))
