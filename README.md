# Launcher Bootstrap

This tool is a bootstrap for your launcher.

Even though it's built for Minecraft, the goal is to be generic-enough to be used for any launcher that requires downloading files and keep them up-to-date, at least for executable / Java / Python apps.

Its what your players will download to run your launcher.

Its goal is to:
1. Download the JVM / Python runtime required by your launcher
2. Update the launcher
3. Start the launcher with the requested runtime

It is made to work with Spectrum and SKCraft. As Spectrum is currently a distant dream it only targets SKCraft.

The bootstrap also features a portable mode that can be started by doing `./bootstrap --path ./launcherfolder`. This will put everything the bootstrap, the launcher then the game is doing in the given folder.

Useful notes:
- The basepath is the "path" argument if it's filled, or the XDG path to `launcher_foldername` otherwise.
- The JVM runtimes are stored at `$basepath/runtime/{component}/{os-arch}`. The launcher can use this folder to store its JVM as long as its in a compatible state. See `jvm_manager.go` if you want to know how they're stored.
- The launcher files are stored at `$basepath/launcher`. This folder is entierly controlled by the bootstrap, don't touch it.

**Note**: While this is made for SKCraft, this won't work properly with the upstream one as it still checks for installed JREs, use [our fork](https://github.com/spectrum-mc/skcraft) for now. [This issue](https://github.com/SKCraft/Launcher/issues/521) relates our effort to upstream it, but for now it's not merged yet.


## Usage

For this guide, we'll make a launcher called "Spectrum Indev", with its related files available on "https://mc.example.com/".

### Setting up the webserver

The webserver needs to have a `launcher_manifest.json` file containing the following:
```json
{
    "version": "v1.0.0",
    "files": [
        {
            "type": "classpath",
            "path": "launcher.jar",
            "hash": "sha256 of the launcher.jar",
            "url": "https://mc.example.com/1.0.0.jar"
        }
    ],
    "main_class": "com.skcraft.launcher.FancyLauncher",
    "args": [
        "--path",
        "${rootPath}"
        "--bootstrap-version",
        "${bsVersion}"
    ],
    "jre": {
        "manifest": "https://launchermeta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json",
        "component": "java-runtime-gamma"
    }
}
```

Lets dig what's going on there.

- `version`: This represents the version of your launcher, this will be used to compare whether the launcher needs to be updated or not.
- `files`: A list of file to download and how they will be used.
- `files.type`: For now, allowed values are: `directory` => A folder will be created at this path, `file` => The file will be downloaded at this path, `classpath` => Same as file but it will be added to the classpath when running a Java application.
- `files.path`: The path where the file should be downloaded relative to the launcher folder.
- `files.hash`: The sha256 of the file, used to re-download it when corrupted / not completely downloaded / tampered with.
- `files.url`: The path to download your file.
- `main_class`: Only useful for Java softwares, this specifies the main class to be run.
- `jre`: The manifest to know where to download Java and which version to use for the launcher.
- `jre.manifest`: The manifest URL. This one is Mojang's one but you should use the [Java Manifest Builder](https://github.com/spectrum-mc/java-manifest-builder) to download them and provide them from your server.
- `jre.component`: The Java version used. Check the JSON in the `manifest` key to find the correct value here.

The `args` key should be an array letting you specify the argument to the launcher to be used. It features special placeholder variables which will be replaced by the bootstrap when running the final command and should be put like this: `${VARIABLE_NAME}`

Here are the allowed values:
| Value | Description |
|-------|-------------|
| rootPath | The path for your launcher to use as its root |
| bsVersion | The bootstrap version |
| isPortable | Is the bootstrap running in portable mode |

### Building the bootstrap

The bootstrap needs to be compiled for every architecture because you need to embed a file that contains the metadata for it.

Lets clone the repository:
```sh
$ git clone git@github.com:spectrum-mc/launcher-bootstrap.git
$ cd launcher-bootstrap
```

Then create a `bs_settings.json` and let's fill it together:
```json
{
	"launcher_manifest": "https://mc.example.com/launcher_manifest.json",
	"launcher_brand": "Spectrum Indev",
	"launcher_foldername": "spectrumlauncher"
}
```

- `launcher_manifest`: should point to the manifest we created in the previous step
- `launcher_brand`: The name displayed everywhere for your launcher
- `launcher_foldername`: The folder name that will be used

N.B. The folder name tries to respect the XDG specs, thus it will store your launcher and its file to `$HOME/.local/share/launchername` on Linux, `@TODO` on OSX and `@TOODO` on Windows.

Please make sure this file is also accessible on `https://mc.example.com/bs_settings.json`. This is not required but if you can't compile one launcher or the other (I'm talking about osx for no particular reason :unamused:) that your user can do it themselves without having to reverse engineer the executable.

Finally compile your executables (@TODO: Maybe make a Github action for this?):
```sh
$ @TODO
```

## ROADMAP

- launcher_manifest.json -> args
- the bs should clear the non wanted files in $basepath/launcher
- store & use the `$basepath/launcher/latest.json` to know when to update
- Implement Python
- Implement generic executable thing
- Multi-"threaded" download (multi-goroutines)

## License

@TODO: Apache2 WITH NOTICE FILE !!
cf: https://opensource.stackexchange.com/questions/8161/license-that-requires-attribution-to-end-users
