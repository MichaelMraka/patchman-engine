from insights import extract, run, rule, make_metadata, make_pass
from insights.parsers import yum_updateinfo
from contextlib import contextmanager
from tempfile import NamedTemporaryFile
import requests

import json
import sys
import os

from kafka import KafkaConsumer, KafkaProducer

KAFKA_ADDRESS = os.getenv("KAFKA_ADDRES", "kafka:9092")
KAFKA_GROUP = os.getenv("KAFKA_GROUP", "patchman")

UPLOAD_TOPIC = os.getenv("UPLOAD_TOPIC", "platform.upload.host-egress")
EVAL_TOPIC = os.getenv("EVAL_TOPIC", "patchman.evaluator.upload")

CONFIG = {
    'bootstrap_servers': KAFKA_ADDRESS,
    'group_id': KAFKA_GROUP,
    'enable_auto_commit': False
}

CONSUMER = KafkaConsumer(UPLOAD_TOPIC, **CONFIG)
PRODUCER = KafkaProducer(**CONFIG)


# Use uname for now, just getting the architecture down
@rule(yum_updateinfo.YumUpdateinfo)
def updateinfo(info):
    return make_pass("updateinfo", info.items)


@contextmanager
def unpacked_remote_archive(url):
    """ Work with remote archive ( download + unpack )"""
    try:
        with NamedTemporaryFile() as file:
            resp = requests.get(url)
            file.write(resp.content)
            file.flush()
            with extract(file.name) as extracted:
                yield extracted
    except BaseException as e:
        print("Error extracting insights archive %s" % e)
    finally:
        pass


def process_archive(path):
    """ Take an unpacked archive, and run our custom rule on it"""
    try:
        parsed = run(updateinfo, root=path)
        result = parsed[updateinfo]
        return result
    except BaseException as e:
        print("Error running insights parser", e)


def process_message(msg):
    """ Process the create/update message from insights, and run whole process on this archive"""
    try:
        msg = json.load(msg)
        with unpacked_remote_archive(msg['platform_metadata']['url']) as unpacked:
            return process_archive(unpacked.tmp_dir)
    except json.JSONDecodeError as e:
        print("Failed to parse message")


def send_evaluation(id, info):
    msg = {'id': id, 'type': 'updates', 'updates': info}
    PRODUCER.send(EVAL_TOPIC, key=id, value=json.dumps(msg))


if __name__ == '__main__':
    CONSUMER.subscribe([UPLOAD_TOPIC])
    for msg in CONSUMER:
        print(msg)
        process_message(msg)

# @contextmanager
# def unpacked_local_archive(path):
#     """ Work with local archive for unpacking, Used for testing"""
#     try:
#         with NamedTemporaryFile() as file:
#             file.write(open(path, 'rb').read())
#             file.flush()
#             with extract(file.name) as extracted:
#                 yield extracted
#     except BaseException as e:
#         print("Error extracting insights archive %s" % e)
#     finally:
#         pass

# if __name__ == "__main__":
#     with unpacked_local_archive(sys.argv[1]) as archive:
#         res = process_archive(archive.tmp_dir)
#         print("Resulted archive evaluation", res)
#
