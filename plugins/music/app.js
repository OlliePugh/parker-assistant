import express from "express";
import "dotenv/config";
import { SpotifyApi } from "@spotify/web-api-ts-sdk";

/**
 * @type {SpotifyApi}
 */
let sdk;

const initialiseSpotify = async () => {
  const refresh_token = process.env.REFRESH_TOKEN;
  const authOptions = {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      Authorization:
        "Basic " +
        Buffer.from(
          process.env.SPOTIFY_CLIENT_ID + ":" + process.env.SPOTIFY_SECRET
        ).toString("base64"),
    },
    body: new URLSearchParams({
      grant_type: "refresh_token",
      refresh_token: refresh_token,
    }),
  };
  const result = await fetch(
    "https://accounts.spotify.com/api/token",
    authOptions
  );

  sdk = SpotifyApi.withAccessToken(
    process.env.SPOTIFY_CLIENT_ID,
    result.json()
  );
};

initialiseSpotify();

const app = express();
const port = 8081;

app.use(express.json());

app.post("/music.devices", async (_, res) => {
  console.log("Getting available devices");
  const devices = await sdk.player.getAvailableDevices();
  // remove unnecessary fields to improve ai processing
  devices.devices.forEach((device) => {
    delete device.is_private_session;
    delete device.is_restricted;
  });

  res.send(devices);
});

app.post("/music.switch_device", async (req, res) => {
  const deviceId = req.body.deviceId;

  if (deviceId === null) {
    console.log("Device id is required");
    return res.status(400).send("Device id is required");
  }

  console.log("transferring playback to device", deviceId);
  try {
    await sdk.player.transferPlayback([deviceId]);
  } catch (e) {
    console.log("Error transferring playback", e);
    res
      .status(401)
      .send(e.message + "use music.devices to get available devices");
    return;
  }
  res.status(200).send({});
});

app.post("/music.play", async (req, res) => {
  const songName = req.body.songName;
  const deviceId = req.body.deviceId;

  console.log(`Playing ${songName} on device ${deviceId}`);
  // return 400 if songName is not provided
  if (!songName) {
    console.log("Song name is required");
    return res.status(400).send("Song name is required");
  }

  const items = await sdk.search(songName, ["track"]);

  if (!items.tracks.items.length) {
    console.log("Song not found");
    return res.status(404).send("Song not found");
  }

  try {
    // await sdk.player.transferPlayback([deviceId]);
    await sdk.player.startResumePlayback(deviceId, null, [
      items.tracks.items[0].uri,
    ]);
  } catch (e) {
    console.log(
      "Error playing music, have you used music.devices to select the ideal device",
      e.message
    );
    res
      .status(401)
      .send(
        `${e.message} have you used music.devices to select the ideal device`
      );
    return;
  }

  console.log(`Successfully started playing ${songName}`);

  res.status(200).send({ songName });
});

app.post("/music.skip", async (req, res) => {
  const skipForward = req.body.skipForward;
  if (skipForward || skipForward === null) {
    sdk.player.skipToNext();
  } else {
    sdk.player.skipToPrevious();
  }

  res.status(200).send({});
});

app.post("/music.continue", async (_, res) => {
  sdk.player.startResumePlayback();

  res.status(200).send({});
});

app.post("/music.stop", async (_, res) => {
  sdk.player.pausePlayback();

  res.status(200).send({});
});

app.listen(port, () => {
  console.log(`Spotify plugin running on http://localhost:${port}`);
});
