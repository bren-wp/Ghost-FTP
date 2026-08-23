#!/usr/bin/env python3
"""Generate and verify the official ByFTP icon and English documentation header."""
from __future__ import annotations
import argparse, binascii, struct, zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ICON_PNG = ROOT / "build" / "icon.png"
ICON_ICO = ROOT / "build" / "icon.ico"
HEADER_PNG = ROOT / "docs" / "images" / "byftp-header.png"
BG=(10,18,32,255); PANEL=(23,39,62,255); TEXT=(238,245,255,255); MUTED=(157,174,196,255); A1=(66,210,196,255); A2=(90,142,255,255)
FONT={
'A':["01110","10001","10001","11111","10001","10001","10001"],
'B':["11110","10001","10001","11110","10001","10001","11110"],
'C':["01111","10000","10000","10000","10000","10000","01111"],
'D':["11110","10001","10001","10001","10001","10001","11110"],
'E':["11111","10000","10000","11110","10000","10000","11111"],
'F':["11111","10000","10000","11110","10000","10000","10000"],
'I':["11111","00100","00100","00100","00100","00100","11111"],
'L':["10000","10000","10000","10000","10000","10000","11111"],
'N':["10001","11001","10101","10011","10001","10001","10001"],
'P':["11110","10001","10001","11110","10000","10000","10000"],
'R':["11110","10001","10001","11110","10100","10010","10001"],
'S':["01111","10000","10000","01110","00001","00001","11110"],
'T':["11111","00100","00100","00100","00100","00100","00100"],
'U':["10001","10001","10001","10001","10001","10001","01110"],
'Y':["10001","10001","01010","00100","00100","00100","00100"],
' ':["00000"]*7,
}

def canvas(w,h):
    p=bytearray(w*h*4)
    for y in range(h):
        t=y/max(1,h-1); c=tuple(round(BG[i]*(1-t)+(19,32,52,255)[i]*t) for i in range(4)); p[y*w*4:(y+1)*w*4]=bytes(c)*w
    return p

def rect(p,w,h,x0,y0,x1,y1,c):
    x0=max(0,x0); y0=max(0,y0); x1=min(w,x1); y1=min(h,y1); row=bytes(c)*max(0,x1-x0)
    for y in range(y0,y1): p[(y*w+x0)*4:(y*w+x1)*4]=row

def rr(p,w,h,x0,y0,x1,y1,r,c):
    for y in range(y0,y1):
        for x in range(x0,x1):
            dx=max(x0+r-x,0,x-(x1-r-1)); dy=max(y0+r-y,0,y-(y1-r-1))
            if dx*dx+dy*dy<=r*r: rect(p,w,h,x,y,x+1,y+1,c)

def line(p,w,h,x0,y0,x1,y1,t,c):
    dx=abs(x1-x0); sx=1 if x0<x1 else -1; dy=-abs(y1-y0); sy=1 if y0<y1 else -1; e=dx+dy
    while True:
        q=max(1,t)//2; rect(p,w,h,x0-q,y0-q,x0+q+1,y0+q+1,c)
        if x0==x1 and y0==y1: break
        e2=2*e
        if e2>=dy: e+=dy; x0+=sx
        if e2<=dx: e+=dx; y0+=sy

def arrow(p,w,h,x0,y,x1,c):
    line(p,w,h,x0,y,x1,y,max(2,w//180),c); d=max(5,abs(x1-x0)//5); s=-1 if x1>x0 else 1
    line(p,w,h,x1,y,x1+s*d,y-d,max(2,w//180),c); line(p,w,h,x1,y,x1+s*d,y+d,max(2,w//180),c)

def icon(size):
    p=canvas(size,size); m=max(2,size//16); rr(p,size,size,m,m,size-m,size-m,max(3,size//7),PANEL)
    pw=max(5,size//4); ph=max(8,size//2); y=(size-ph)//2; lx=size//8; rx=size-size//8-pw
    rr(p,size,size,lx,y,lx+pw,y+ph,max(2,size//32),TEXT); rr(p,size,size,rx,y,rx+pw,y+ph,max(2,size//32),TEXT)
    for yy in (y+ph//4,y+ph//2,y+3*ph//4): rect(p,size,size,lx+pw//5,yy,lx+4*pw//5,yy+max(1,size//40),BG); rect(p,size,size,rx+pw//5,yy,rx+4*pw//5,yy+max(1,size//40),BG)
    arrow(p,size,size,lx+pw+size//20,size//2-size//10,rx-size//20,A1); arrow(p,size,size,rx-size//20,size//2+size//10,lx+pw+size//20,A2)
    return size,size,p

def text(p,w,h,s,x,y,scale,c):
    cur=x
    for ch in s.upper():
        for gy,row in enumerate(FONT.get(ch,FONT[' '])):
            for gx,b in enumerate(row):
                if b=='1': rect(p,w,h,cur+gx*scale,y+gy*scale,cur+(gx+1)*scale,y+(gy+1)*scale,c)
        cur+=6*scale

def header():
    w,h=1200,320; p=canvas(w,h); rr(p,w,h,40,40,280,280,42,PANEL); iw,ih,ip=icon(200)
    for y in range(ih): p[((60+y)*w+60)*4:((60+y)*w+60+iw)*4]=ip[y*iw*4:(y+1)*iw*4]
    text(p,w,h,'BYFTP',335,72,17,TEXT); text(p,w,h,'SECURE FILE TRANSFER',338,215,5,MUTED); rect(p,w,h,338,270,1095,276,A1)
    return w,h,p

def png(w,h,p):
    def chunk(k,d):
        b=k+d; return struct.pack('>I',len(d))+b+struct.pack('>I',binascii.crc32(b)&0xffffffff)
    raw=bytearray()
    for y in range(h): raw.append(0); raw.extend(p[y*w*4:(y+1)*w*4])
    return b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',w,h,8,6,0,0,0))+chunk(b'IDAT',zlib.compress(bytes(raw),9))+chunk(b'IEND',b'')

def ico():
    sizes=(16,24,32,48,64,128,256); imgs=[]
    for n in sizes: w,h,p=icon(n); imgs.append(png(w,h,p))
    off=6+16*len(imgs); entries=[]
    for n,d in zip(sizes,imgs): q=0 if n==256 else n; entries.append(struct.pack('<BBBBHHII',q,q,0,0,1,32,len(d),off)); off+=len(d)
    return struct.pack('<HHH',0,1,len(imgs))+b''.join(entries)+b''.join(imgs)

def expected():
    w,h,p=icon(512); hw,hh,hp=header(); return {ICON_PNG:png(w,h,p),ICON_ICO:ico(),HEADER_PNG:png(hw,hh,hp)}

def main():
    ap=argparse.ArgumentParser(description='Generate deterministic ByFTP brand assets'); ap.add_argument('--check',action='store_true'); args=ap.parse_args(); exp=expected()
    if args.check:
        bad=[str(p.relative_to(ROOT)) for p,d in exp.items() if not p.is_file() or p.read_bytes()!=d]
        if bad: raise SystemExit('Brand assets are out of date: '+', '.join(bad))
        print('BRAND_ASSETS=PASS'); return 0
    for p,d in exp.items(): p.parent.mkdir(parents=True,exist_ok=True); p.write_bytes(d); print('Updated:',p.relative_to(ROOT))
    print('BRAND_ASSETS=PASS'); return 0
if __name__=='__main__': raise SystemExit(main())
