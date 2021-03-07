import json
import requests

url = "https://tartanhackathon.uc.r.appspot.com/recommend"
sample_text = '''
My point is that writing a new operating system that is closely tied to any
particular piece of hardware, especially a weird one like the Intel line,
is basically wrong.  An OS itself should be easily portable to new hardware
platforms.  When OS/360 was written in assembler for the IBM 360
25 years ago, they probably could be excused.  When MS-DOS was written
specifically for the 8088 ten years ago, this was less than brilliant, as
IBM and Microsoft now only too painfully realize. Writing a new OS only for the
386 in 1991 gets you your second 'F' for this term.  But if you do real well
on the final exam, you can still pass the course.
'''

# the body of the request is a json
query = {
    "texts": [
        {
            "body": sample_text
        }
    ]
}
# url parameters specify the number of labels you want to query on
params = {"num": 10}

res = requests.post(url, json=query, params=params)
print(res.json())
