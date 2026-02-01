#!/usr/bin/env python3
"""
MediaPipe hand detection service for v0.10+ (Task API).
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
import os

# MediaPipe 0.10+ Task API
import mediapipe as mp
from mediapipe.tasks.python.vision import HandLandmarker, HandLandmarkerOptions, RunningMode
from mediapipe.tasks.python.core.base_options import BaseOptions

# Find model file
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
MODEL_PATH = os.path.join(SCRIPT_DIR, 'hand_landmarker.task')

if not os.path.exists(MODEL_PATH):
    print(f"Error: Model file not found at {MODEL_PATH}", file=sys.stderr)
    sys.exit(1)

def main():
    # Configure options
    base_options = BaseOptions(model_asset_path=MODEL_PATH)
    options = HandLandmarkerOptions(
        base_options=base_options,
        num_hands=2,
        min_hand_detection_confidence=0.5,
        min_hand_presence_confidence=0.5,
        min_tracking_confidence=0.5,
        running_mode=RunningMode.IMAGE
    )
    
    # Create detector
    detector = HandLandmarker.create_from_options(options)

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
        image_bgr = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if image_bgr is None:
            print(json.dumps({"hands": []}), flush=True)
            continue

        # Convert BGR to RGB
        image_rgb = cv2.cvtColor(image_bgr, cv2.COLOR_BGR2RGB)
        
        # Create MediaPipe Image
        mp_image = mp.Image(image_format=mp.ImageFormat.SRGB, data=image_rgb)

        # Detect
        detection_result = detector.detect(mp_image)

        # Format output
        output = {"hands": []}

        if detection_result.hand_landmarks:
            for i, hand_landmarks in enumerate(detection_result.hand_landmarks):
                handedness = "Right"
                score = 0.9
                
                if detection_result.handedness and i < len(detection_result.handedness):
                    handedness_info = detection_result.handedness[i][0]
                    handedness = handedness_info.category_name
                    score = handedness_info.score

                points = []
                for lm in hand_landmarks:
                    points.append({
                        "x": lm.x,
                        "y": lm.y,
                        "z": lm.z
                    })

                output["hands"].append({
                    "points": points,
                    "handedness": handedness,
                    "score": score
                })

        print(json.dumps(output), flush=True)


if __name__ == "__main__":
    main()
