import express from "express";
import "dotenv/config";
import { SpotifyApi } from "@spotify/web-api-ts-sdk";

const sdk = SpotifyApi.withAccessToken(process.env.SPOTIFY_CLIENT_ID, {
  access_token: process.env.ACCESS_TOKEN,
  token_type: "Bearer",
  refresh_token: process.env.REFRESH_TOKEN,
});

const app = express();
const port = 8081;

app.use(express.json());

app.post("/music.devices", async (_, res) => {
  const devices = await sdk.player.getAvailableDevices();
  // remove unnecessary fields
  devices.devices.forEach((device) => {
    delete device.is_active;
    delete device.is_private_session;
    delete device.is_restricted;
    delete device.volume_percent;
    delete device.supports_volume;
  });

  res.send(devices);
});

app.post("/music.play", async (req, res) => {
  const songName = req.body.songName;
  console.log(`Playing ${songName}`);
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

  await sdk.player.addItemToPlaybackQueue(items.tracks.items[0].uri);
  await sdk.player.skipToNext();

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
