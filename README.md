# Stream WebRTC -> linux v4l2 virtual camera 

Подключается к webrtc потоку от браузера и запихивает видеопоток от браузера в виртуальную вебкамеру, которую можно 
посмотреть с помощью ffplay и тп.

Изначальная идея (оригинального автора) в том, чтобы использовать камеру телефона, как локальную камеру linux с помощью 
webrtc.

Технологии:

* Golang
* Pion (webrtc)
* gstreamer
* v4l2 (virtual webcam)

# Как запустить (инструкция ещё не готова)

0. Устанавливаем gstreamer, gstreamer libav (ffmpeg), v4l2loopback
1. Настраиваем вирутальную камеру
```shell
sudo modprobe v4l2loopback video_nr="42"\ 'card_label=virtcam'\ exclusive_caps=1 max_buffers=2
v4l2-ctl -d /dev/video42 --all

# for reset:
#modprobe --remove v4l2loopback
```

2. Запускаем
```shell
go build -o webrtc ./cmd && GST_DEBUG=*:2 ./webrtc 
```
3. Переходим на 127.0.0.1:8888 и нажимает start (после того как в первом поле появится base64 токен)
4. смотрим виртуальную камеру: 
```shell
ffplay /dev/video42
```