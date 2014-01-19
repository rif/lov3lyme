#!/usr/bin/env python3

import re
import os
import json
import glob
from collections import OrderedDict
from os.path import join, getsize, isdir

regex = re.compile("trans\(\"(.+?)\",\s?ctx\)|\{\{\s?trans\s?\"(.+?)\"\s?\.ctx\s?\}\}")
strings = []
for root, dirs, files in os.walk('.'):    
    if 'langs' in dirs: dirs.remove('langs') # dont visit langs dir    
    for f in (join(root, name) for name in files):    
        with open(f) as fp:            
            for m1, m2 in regex.findall(fp.read()):
                if m1.strip() != '': strings.append(m1)
                if m2.strip() != '': strings.append(m2)
strings = list(set(strings))
for lang in glob.glob("langs/*"):
    if isdir(join('langs',lang)): continue
    m = {}
    with open(lang) as fp: 
        m = json.loads(fp.read())              
    for s in strings:
        if s not in m:
            m[s]=""
    with open(lang, 'w') as fp:        
        dump = json.dumps(OrderedDict(sorted(m.items(), key=lambda t: t[0])), indent=1, ensure_ascii=False)
        fp.write(dump)
