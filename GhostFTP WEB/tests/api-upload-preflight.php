<?php
declare(strict_types=1);

$source = file_get_contents(__DIR__ . '/../api.php');
if (!is_string($source)) {
    fwrite(STDERR, "FAIL: unable to inspect API upload source.\n");
    exit(1);
}
$start = strpos($source, "case 'upload':");
$end = $start !== false ? strpos($source, "throw new RuntimeException('Nepoznata akcija.')", $start) : false;
if ($start === false || $end === false || $end <= $start) {
    fwrite(STDERR, "FAIL: unable to isolate upload API branch.\n");
    exit(1);
}
$block = substr($source, $start, $end - $start);
$planPos = strpos($block, '$uploadPlan = [];');
$validationLoopPos = strpos($block, 'foreach ($names as $i => $original)');
$shapePos = strpos($block, 'Upload zahtjev sadrži neusklađene metapodatke datoteka.');
$duplicateRemotePos = strpos($block, 'Više upload datoteka ne smije ciljati istu udaljenu putanju.');
$appendPos = strpos($block, '$uploadPlan[] =');
$executionLoopPos = strpos($block, 'foreach ($uploadPlan as $item)');
$uploadPos = strpos($block, 'uploadWithConflict(', $executionLoopPos === false ? 0 : $executionLoopPos);
if (
    $planPos === false || $validationLoopPos === false || $shapePos === false
    || $duplicateRemotePos === false || $appendPos === false
    || $executionLoopPos === false || $uploadPos === false
    || !($shapePos < $validationLoopPos
        && $planPos < $validationLoopPos
        && $validationLoopPos < $duplicateRemotePos
        && $duplicateRemotePos < $appendPos
        && $appendPos < $executionLoopPos
        && $executionLoopPos < $uploadPos)
) {
    fwrite(STDERR, "FAIL: upload request is not fully preflighted before remote mutation.\n");
    exit(1);
}

echo "WEB_API_UPLOAD_PREFLIGHT_TEST=PASS\n";
