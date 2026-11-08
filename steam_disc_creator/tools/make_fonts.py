#!/usr/bin/env python3
# Erzeugt die kleinen, eingebetteten Schrift-Teilmengen (WOFF) aus den Original-Schriften des Physical Release Studio
# (assets/fonts/Inter und assets/fonts/JetBrainsMono, Lizenz: SIL OFL 1.1). Nur nötig, wenn sich die Schriften ändern;
# die erzeugten Dateien liegen in src/fonts/ und werden von build.py eingebettet. Benötigt: pip install fonttools
import sys, os, io
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer
from fontTools import subset

UNICODES = (list(range(0x20, 0x7F)) + list(range(0xA0, 0x180)) +
            [0x2010, 0x2011, 0x2012, 0x2013, 0x2014, 0x2018, 0x2019, 0x201A, 0x201C, 0x201D, 0x201E, 0x2022, 0x2026, 0x2039, 0x203A,
             0x20AC, 0x2190, 0x2191, 0x2192, 0x2193, 0x2197, 0x2212, 0x00D7, 0x2713])

def make(src, dst, limits):
    f = TTFont(src)
    f = instancer.instantiateVariableFont(f, limits)
    buf = io.BytesIO(); f.save(buf); buf.seek(0); f = TTFont(buf)  # neu laden: vermeidet Fehler beim Teilmengen-Bilden
    opt = subset.Options()
    opt.flavor = 'woff'
    opt.layout_features = ['kern', 'liga', 'calt', 'ccmp', 'locl', 'mark', 'mkmk']
    opt.name_IDs = [1, 2, 3, 4, 6]
    opt.notdef_outline = True
    s = subset.Subsetter(opt)
    s.populate(unicodes=UNICODES)
    s.subset(f)
    f.flavor = 'woff'
    f.save(dst)
    print(dst, os.path.getsize(dst))

if __name__ == '__main__':
    repo = sys.argv[1]  # Wurzel des Physical-Release-Studio-Repos
    out = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'src', 'fonts')
    os.makedirs(out, exist_ok=True)
    make(os.path.join(repo, 'assets/fonts/Inter/Inter-VariableFont_opsz,wght.ttf'), os.path.join(out, 'Inter-subset.woff'), {'opsz': 14, 'wght': (400, 800)})
    make(os.path.join(repo, 'assets/fonts/JetBrainsMono/JetBrainsMono-VariableFont_wght.ttf'), os.path.join(out, 'JetBrainsMono-subset.woff'), {'wght': (400, 700)})
