```shell
sudo modprobe v4l2loopback video_nr="42"\ 'card_label=virtcam'\ exclusive_caps=1 max_buffers=2
v4l2-ctl -d /dev/video42 --all
modprobe --remove v4l2loopback
```