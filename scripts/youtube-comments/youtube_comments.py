#!/usr/bin/env python3
from collections import Counter
from urllib.parse import urlparse, parse_qs
from math import inf
from wordcloud import WordCloud
from pathlib import Path
import csv
import json
import googleapiclient.discovery

CREDENTIALS_PATH = 'creds.json'
FONT_PATH = 'NotoSansCJKjp-Regular.otf'
STOP_WORDS_PATH = 'stopwords-ja.json'


def parse_comments(comment_items: list):
    """
        parse comments from raw_comments
    """
    for item in comment_items:
        snippet = item['snippet']
        comment = snippet.get('topLevelComment') or item
        comment_snippet = comment['snippet']
        replies = item.get('replies')
        reply_comments = replies['comments'] if replies else list()
        yield {
            'id': item['id'],
            'author': comment_snippet['authorDisplayName'],
            'profile_image_url': comment_snippet['authorProfileImageUrl'],
            'text': comment_snippet['textDisplay'],
            'raw_text': comment_snippet['textOriginal'],
            'likes': comment_snippet['likeCount'],
            'published_at': comment_snippet['publishedAt'],
            'updated_at': comment_snippet['updatedAt'],
            'num_replies': snippet.get('totalReplyCount', 0),
            'replies': reply_comments,
        }


def parse_comments_and_replies(comments):
    """
        parse comments and replies from raw_comments
    """
    comments_and_replies = list()
    for comment in parse_comments(comments):
        comments_and_replies.append(comment)
        replies = parse_comments(comment['replies'])
        comments_and_replies.extend(replies)
    return comments_and_replies


def video_id_from_url(video_url: str) -> str:
    """
        get the video id from a youtube url
    """
    url = urlparse(video_url)
    if url.scheme != "https" or url.netloc != 'www.youtube.com':
        raise Exception("invalid url: {}".format(video_url))

    params = parse_qs(url.query)
    if 'v' not in params:
        raise Exception("missing video id in url: {}".format(video_url))

    video_id = params.get('v')[0]

    return video_id


def fetch_comments(video_url: str, max_pages=inf) -> list:
    """
        fetch comment data from video at given url
        gives raw_comments
    """
    if max_pages <= 0:
        return list()

    video_id = video_id_from_url(video_url)

    api_service_name = "youtube"
    api_version = "v3"
    with open(CREDENTIALS_PATH) as f:
        creds = json.load(f)
        api_key = creds.get('youtube-comments-api-key')

    youtube = googleapiclient.discovery.build(
        api_service_name, api_version, developerKey=api_key,
        cache_discovery=False)

    req = youtube.commentThreads().list(
        part="snippet,replies",
        videoId=video_id)
    res = req.execute()

    page_no = 1
    raw_comments = list()
    while True:
        print('[get_comments] page:', page_no)
        raw_comments.extend(res['items'])
        # check for next page
        token = res.get('nextPageToken')
        if token and page_no < max_pages:
            req = youtube.commentThreads().list(
                part="snippet,replies",
                videoId=video_id,
                pageToken=token)
            res = req.execute()
            page_no += 1
        else:
            break

    return raw_comments


def save_comments(raw_comments, parsed_comments, save_as='comments.tsv'):
    """
        save raw_comments to json lines file
        save parsed_comments to a tsv file
    """
    jsonl_path = Path(save_as).with_suffix('.jsonl')
    csv_path = Path(save_as).with_suffix('.tsv')
    csv_fields = [
        'id',
        'author',
        'text',
        'raw_text',
        'likes',
        'published_at',
        'updated_at',
        'num_replies',
    ]
    with open(jsonl_path, 'w') as jsonlf, open(csv_path, 'w') as csvf:
        print("[save_comments] saving to '{}'".format(jsonl_path))
        for c in raw_comments:
            jsonlf.write('{}\n'.format(json.dumps(c)))
        print("[save_comments] saving to '{}'".format(csv_path))
        csv_writer = csv.DictWriter(csvf, csv_fields,
                                    delimiter='\t', extrasaction='ignore')
        csv_writer.writeheader()
        csv_writer.writerows(parsed_comments)


def mentions(comments, keywords) -> list:
    return [c for c in comments
            if any(k in c['raw_text'] for k in keywords)]


def word_counts(comments) -> list:
    """
        count words in all parsed commments
    """
    import nagisa
    with open(STOP_WORDS_PATH) as f:
        stopwords = json.load(f)
    words = Counter()
    for text in [c['raw_text'] for c in comments]:
        t = nagisa.extract(text, extract_postags=[
            '名詞', '代名詞', '形容詞'])
        for w in set(t.words):
            if w.isalpha() and w not in stopwords:
                words[w] += 1
    return words


def create_word_cloud(words, save_as="word-cloud.png",):
    """
        create word cloud from parsed comments and save it
    """
    wc = WordCloud(
        width=1024, height=512,
        background_color='white',
        relative_scaling=0.8,
        min_font_size=10,
        max_words=64,
        font_path=FONT_PATH).generate_from_frequencies(words)
    if save_as:
        save_as = str(save_as)
        print("[create_word_cloud] saving to '{}'".format(save_as))
        wc.to_file(save_as)
    return wc


def main(url, save_to, max_pages):
    from datetime import datetime, timezone
    Path(save_to).mkdir(parents=True, exist_ok=True)
    print("saving to '{}' max_pages={}".format(save_to, max_pages))
    timestamp = datetime.now(timezone.utc).astimezone().isoformat()
    raw_comments = fetch_comments(url, max_pages=max_pages)
    parsed_comments = parse_comments_and_replies(raw_comments)

    # save data
    comments_path = Path(save_to) / 'comments-{}.tsv'.format(timestamp)
    words_path = Path(save_to) / 'words-{}.txt'.format(timestamp)
    wordcloud_path = Path(save_to) / 'word-cloud-{}.png'.format(timestamp)
    words = word_counts(parsed_comments)
    print("saving to '{}'".format(words_path))
    with open(words_path, 'w') as f:
        for w, c in words.most_common():
            f.write("{} {}\n".format(w, c))
    create_word_cloud(words, save_as=wordcloud_path)
    save_comments(raw_comments, parsed_comments, save_as=comments_path)


if __name__ == '__main__':
    import sys
    if len(sys.argv) != 2 and len(sys.argv) != 3:
        print("usage: {} [video_url] [max_pages=inf]".format(sys.argv[0]))
        sys.exit(0)
    url = sys.argv[1]
    max_pages = int(sys.argv[2]) if len(sys.argv) == 3 else inf
    video_id = video_id_from_url(url)
    save_to = Path('data') / video_id
    main(url, save_to, max_pages=max_pages)
