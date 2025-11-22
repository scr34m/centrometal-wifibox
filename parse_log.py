import re
import json
from collections import defaultdict

values_per_key = defaultdict(set)

with open("stdout.log", "r", encoding="utf-8") as f:
    for line in f:
        match = re.search(r'\{.*?\}', line)
        if match:
            json_text = match.group(0).strip()
            try:
                data = json.loads(json_text)
                for k, v in data.items():
                    values_per_key[k].add(v)
            except json.JSONDecodeError:
                continue

for key in sorted(values_per_key.keys()):
    unique_values = values_per_key[key]
    count = len(unique_values)
    print(f"{key}: {count} unique values", end="")
    if count < 11:
        print(f" -> {sorted(unique_values)}")
    else:
        print()
