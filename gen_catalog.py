import json
import re

with open(r'd:\PROJECTS\acts\src\js\app.js', 'r', encoding='utf-8') as f:
    content = f.read()

m = re.search(r'var PRODUCTS = (\[[\s\S]*?\]);\s*var SERVICES', content)
products = json.loads(m.group(1))

out = []
for p in products:
    price_clean = int(re.sub(r'[^\d]', '', p.get('price', '0')) or 0)
    name_lower = p.get('name', '').lower()
    brand = 'BBIA'
    if 'about tone' in name_lower or 'eglips' in name_lower:
        brand = 'ABOUT TONE'
    elif 'hince' in name_lower:
        brand = 'HINCE'
    elif 'mixsoon' in name_lower:
        brand = 'MIXSOON'
    elif 'sky' in name_lower:
        brand = 'SKY'
        
    keywords = [brand.lower(), p.get('cat', '')]
    for w in name_lower.replace('-', ' ').replace('/', ' ').split():
        if len(w) > 2 and w not in keywords:
            keywords.append(w)
            
    for v in p.get('variants', []):
        code = v.get('code', '').lower()
        if code and code not in keywords:
            keywords.append(code)
            
    kw_str = '[]string{' + ', '.join([f'"{k}"' for k in keywords[:8]]) + '}'
    
    code = f"""\t\t{{
\t\t\tID: {p['id']}, SkyID: {p['sky_id']}, Category: "{p['cat']}", Brand: "{brand}",
\t\t\tName: "{p['name']}", Desc: "{p['desc']}",
\t\t\tPrice: {price_clean}, PriceDisplay: "{p['price']}", StockQuantity: 50,
\t\t\tImageUrl:  "{p.get('img', '')}",
\t\t\tUndertone: "neutral", Keywords: {kw_str},
\t\t}},"""
    out.append(code)

with open(r'd:\PROJECTS\acts-backend-go\internal\database\catalog_gen.txt', 'w', encoding='utf-8') as g:
    g.write('\n'.join(out))
print('SUCCESS_COUNT:', len(out))

