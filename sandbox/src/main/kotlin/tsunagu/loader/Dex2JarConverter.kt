package tsunagu.loader

import com.googlecode.d2j.dex.Dex2jar
import com.googlecode.d2j.reader.MultiDexFileReader
import com.googlecode.dex2jar.tools.BaksmaliBaseDexExceptionHandler
import java.io.File
import java.nio.file.Files

class Dex2JarConversionException(message: String) : Exception(message)

object Dex2JarConverter {
    /**
     * Convert an APK's dex bytecode to a jar.
     *
     * Mirrors Suwayomi's PackageTools.dex2jar invocation (same underlying
     * dex2jar library, non-default flags). The default-flags call
     * (Dex2jar.from(apkFile).to(outputJar)) produces structurally-plausible
     * but incorrect bytecode for some Kotlin-generated classes (e.g.
     * kotlinx.serialization's $$serializer classes), surfacing later as
     * IncompatibleClassChangeError at runtime instead of a conversion-time
     * failure. reUseReg(false) and topoLogicalSort() avoid the bad register
     * reconstruction; dontSanitizeNames(true) preserves Kotlin's special
     * characters (e.g. the "$$" in generated serializer class names).
     */
    fun convert(apkFile: File): File {
        val outputJar = Files.createTempFile("tsunagu-ext-", ".jar")
        Files.deleteIfExists(outputJar)

        val reader = MultiDexFileReader.open(apkFile.readBytes())
        val handler = BaksmaliBaseDexExceptionHandler()

        Dex2jar
            .from(reader)
            .withExceptionHandler(handler)
            .reUseReg(false)
            .topoLogicalSort()
            .skipDebug(true)
            .optimizeSynchronized(false)
            .printIR(false)
            .noCode(false)
            .skipExceptions(false)
            .dontSanitizeNames(true)
            .to(outputJar)

        if (handler.hasException()) {
            val errorFile = Files.createTempFile("tsunagu-ext-error-", ".txt")
            handler.dump(errorFile, emptyArray<String>())
            throw Dex2JarConversionException(
                "dex2jar reported conversion errors for ${apkFile.name}; details: $errorFile"
            )
        }

        BytecodeEditor.fixAndroidClasses(outputJar)

        return outputJar.toFile()
    }
}
