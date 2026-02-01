#!/usr/bin/env python3
"""
MediaPipe hand detection service.
Reads JPEG frames from stdin, outputs JSON landmarks to stdout.

Protocol:
- Input: 4-byte length (big-endian) + JPEG bytes
- Output: JSON line with landmarks array
"""

import sys
import struct
import json
import cv2
import numpy as np
import mediapipe as mp

mp_hands = mp.solutions.hands


def main():
    hands = mp_hands.Hands(
        static_image_mode=False,
        max_num_hands=2,
        min_detection_confidence=0.5,
        min_tracking_confidence=0.5
    )

    while True:
        # Read frame length (4 bytes, big-endian)
        length_bytes = sys.stdin.buffer.read(4)
        if len(length_bytes) < 4:
            break

        length = struct.unpack('>I', length_bytes)[0]

        # Read JPEG data
        jpeg_data = sys.stdin.buffer.read(length)
        if len(jpeg_data) < length:
            break

        # Decode image
        nparr = np.frombuffer(jpeg_data, np.uint8)
        image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if image is None:
            print(json.dumps({"hands": []}), flush=True)
            continue

        # Convert BGR to RGB
        image_rgb = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)

        # Process
        results = hands.process(image_rgb)

        # Format output
        output = {"hands": []}

        if results.multi_hand_landmarks:
            for i, hand_landmarks in enumerate(results.multi_hand_landmarks):
                handedness = "Right"
                if results.multi_handedness:
                    handedness = results.multi_handedness[i].classification[0].label

                points = []
                for lm in hand_landmarks.landmark:
                    points.append({
                        "x": lm.x,
                        "y": lm.y,
                        "z": lm.z
                    })

                score = 0.9
                if results.multi_handedness:
                    score = results.multi_handedness[i].classification[0].score

                output["hands"].append({
                    "points": points,
                    "handedness": handedness,
                    "score": score
                })

        print(json.dumps(output), flush=True)

    hands.close()


if __name__ == "__main__":
    main()
